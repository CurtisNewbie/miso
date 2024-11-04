package miso

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/version"
)

const (
	// Default shutdown hook execution order.
	DefShutdownOrder = 5

	// Components like database that are essential and must be ready before anything else.
	BootstrapOrderL1 = -20

	// Components that are bootstraped before the web server, such as metrics stuff.
	BootstrapOrderL2 = -15

	// The web server or anything similar, bootstraping web server doesn't really mean that we will receive inbound requests.
	BootstrapOrderL3 = -10

	// Components that introduce inbound requests or job scheduling.
	//
	// When these components bootstrap, the server is considered truly running.
	// For example, service registration (for service discovery), MQ broker connection and so on.
	BootstrapOrderL4 = -5
)

var (
	loggerOut    io.Writer = os.Stdout
	loggerErrOut io.Writer = os.Stderr

	globalApp *MisoApp = newApp()
)

func init() {
	SetDefProp(PropServerGracefulShutdownTimeSec, 0)
}

type MisoApp struct {
	rail         Rail
	configLoaded bool

	// channel for signaling server shutdown
	manualSigQuit chan int

	shuttingDown   bool
	shutingDownRwm sync.RWMutex // rwmutex for shuttingDown

	shutdownHook []OrderedShutdownHook
	shmu         sync.Mutex // mutex for shutdownHook

	serverBootrapCallbacks      []ComponentBootstrap
	preServerBootstrapListener  []func(r Rail) error
	postServerBootstrapListener []func(r Rail) error

	store *util.RWMap[string, any]
}

// Get global miso app.
func App() *MisoApp {
	return globalApp
}

func newApp() *MisoApp {
	return &MisoApp{
		manualSigQuit: make(chan int, 1),
		rail:          EmptyRail(),
		configLoaded:  false,
		shuttingDown:  false,
		store:         util.NewRWMap[string, any](),
	}
}

func (a *MisoApp) Get(k string) any {
	v, _ := a.store.Get(k)
	return v
}

func (a *MisoApp) GetElse(k string, elseFunc func(k string) any) any {
	v, _ := a.store.GetElse(k, elseFunc)
	return v
}

func (a *MisoApp) Put(k string, v any) {
	a.store.Put(k, v)
}

// Bootstrap miso app.
func (a *MisoApp) Bootstrap(args []string) {
	a.LoadConfig(args)

	osSigQuit := make(chan os.Signal, 2)
	signal.Notify(osSigQuit, os.Interrupt, syscall.SIGTERM)

	a.AddOrderedShutdownHook(0, MarkServerShuttingDown) // the first hook to be called
	var rail Rail = a.rail

	start := time.Now().UnixMilli()
	defer a.triggerShutdownHook()

	appName := GetPropStr(PropAppName)
	if appName == "" {
		rail.Fatalf("Property '%s' is required", PropAppName)
	}

	rail.Infof("\n\n---------------------------------------------- starting %s -------------------------------------------------------\n", appName)
	rail.Infof("Miso Version: %s", version.Version)
	rail.Infof("Production Mode: %v", GetPropBool(PropProdMode))

	// invoke callbacks to setup server, sometime we need to setup stuff right after the configuration being loaded
	if e := a.callPreServerBootstrapListeners(rail); e != nil {
		rail.Errorf("Error occurred while invoking pre server bootstrap callbacks, %v", e)
		return
	}

	// bootstrap components, these are sorted by their orders
	sort.Slice(a.serverBootrapCallbacks, func(i, j int) bool { return a.serverBootrapCallbacks[i].Order < a.serverBootrapCallbacks[j].Order })
	Debugf("serverBootrapCallbacks: %+v", a.serverBootrapCallbacks)
	for _, sbc := range a.serverBootrapCallbacks {
		if sbc.Condition != nil {
			ok, ce := sbc.Condition(rail)
			if ce != nil {
				rail.Errorf("Failed to bootstrap server component: %v, failed on condition check, %v", sbc.Name, ce)
				return
			}
			if !ok {
				continue
			}
		}

		start := time.Now()
		if e := sbc.Bootstrap(rail); e != nil {
			rail.Errorf("Failed to bootstrap server component: %v, %v", sbc.Name, e)
			return
		}
		took := time.Since(start)
		rail.Debugf("Callback %-30s - took %v", sbc.Name, took)
		if took >= 5*time.Second {
			rail.Warnf("Component '%s' might be too slow to bootstrap, took: %v", sbc.Name, took)
		}
	}
	a.serverBootrapCallbacks = nil

	end := time.Now().UnixMilli()
	rail.Infof("\n\n---------------------------------------------- %s started (took: %dms) --------------------------------------------\n", appName, end-start)

	// invoke listener for serverBootstraped event
	if e := a.callPostServerBootstrapListeners(rail); e != nil {
		rail.Errorf("Error occurred while invoking post server bootstrap callbacks, %v", e)
		return
	}

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	select {
	case sig := <-osSigQuit:
		rail.Infof("Received OS signal: %v, exiting", sig)
	case <-a.manualSigQuit: // or wait for maunal shutdown signal
		rail.Infof("Received manual shutdown signal, exiting")
	}

}

// Load app configuration.
func (a *MisoApp) LoadConfig(args []string) {
	if a.configLoaded {
		return
	}

	// default way to load configuration
	DefaultReadConfig(args, a.rail)

	if err := ConfigureLogging(a.rail); err != nil {
		a.rail.Errorf("Configure logging failed, %v", err)
	}
	a.configLoaded = true
}

// Trigger shutdown hook
func (a *MisoApp) triggerShutdownHook() {
	timeout := GetPropInt(PropServerGracefulShutdownTimeSec)

	f := util.RunAsync(func() (any, error) {
		a.shmu.Lock()
		defer a.shmu.Unlock()

		sort.Slice(a.shutdownHook, func(i, j int) bool { return a.shutdownHook[i].Order < a.shutdownHook[j].Order })
		for _, hook := range a.shutdownHook {
			hook.Hook()
		}
		return nil, nil
	})
	if timeout > 0 {
		timeoutDur := time.Duration(timeout * int(time.Second))
		_, err := f.TimedGet(int(timeoutDur / time.Millisecond))
		if errors.Is(err, util.ErrGetTimeout) {
			a.rail.Infof("Exceeded server graceful shutdown period (%v), stop waiting for shutdown hook execution", timeoutDur)
		}
	} else {
		_, err := f.Get()
		if err != nil {
			a.rail.Infof("Unexpected error occurred while executing shutdown hooks, %v", err)
		}
	}

}

func (a *MisoApp) AddOrderedShutdownHook(order int, hook func()) {
	a.shmu.Lock()
	defer a.shmu.Unlock()
	a.shutdownHook = append(a.shutdownHook, OrderedShutdownHook{
		Order: order,
		Hook:  hook,
	})
}

// check if the server is shutting down
func (a *MisoApp) IsShuttingDown() bool {
	a.shutingDownRwm.RLock()
	defer a.shutingDownRwm.RUnlock()
	return a.shuttingDown
}

// mark that the server is shutting down
func (a *MisoApp) MarkServerShuttingDown() {
	a.shutingDownRwm.Lock()
	defer a.shutingDownRwm.Unlock()
	a.shuttingDown = true
}

// Shutdown server
func (a *MisoApp) Shutdown() {
	a.manualSigQuit <- 1
}

func (a *MisoApp) callPostServerBootstrapListeners(rail Rail) error {
	i := 0
	for i < len(a.postServerBootstrapListener) {
		if e := a.postServerBootstrapListener[i](rail); e != nil {
			return e
		}
		i++
	}
	a.postServerBootstrapListener = nil
	return nil
}

func (a *MisoApp) callPreServerBootstrapListeners(rail Rail) error {
	i := 0
	for i < len(a.preServerBootstrapListener) {
		if e := a.preServerBootstrapListener[i](rail); e != nil {
			return e
		}
		i++
	}
	a.preServerBootstrapListener = nil
	return nil
}

// Register server component bootstrap callback
//
// When such callback is invoked, configuration should be fully loaded, the callback is free to read the loaded configuration
// and decide whether or not the server component should be initialized, e.g., by checking if the enable flag is true.
func (a *MisoApp) RegisterBootstrapCallback(bootstrapComponent ComponentBootstrap) {
	a.serverBootrapCallbacks = append(a.serverBootrapCallbacks, bootstrapComponent)
}

// Add listener that is invoked when server is finally bootstrapped
//
// This usually means all server components are started, such as MySQL connection, Redis Connection and so on.
//
// Caller is free to call PostServerBootstrapped inside another PostServerBootstrapped callback.
func (a *MisoApp) PostServerBootstrapped(callback func(rail Rail) error) {
	if callback == nil {
		return
	}
	a.postServerBootstrapListener = append(a.postServerBootstrapListener, callback)
}

// Add listener that is invoked before the server is fully bootstrapped
//
// This usually means that the configuration is loaded, and the logging is configured, but the server components are not yet initialized.
//
// Caller is free to call PostServerBootstrapped or PreServerBootstrap inside another PreServerBootstrap callback.
func (a *MisoApp) PreServerBootstrap(callback func(rail Rail) error) {
	if callback == nil {
		return
	}
	a.preServerBootstrapListener = append(a.preServerBootstrapListener, callback)
}

/*
Bootstrap server

This func will attempt to create http server, connect to MySQL, Redis or Consul based on the configuration loaded.

It also handles service registration/de-registration on Consul before Gin bootstraped and after
SIGTERM/INTERRUPT signals are received.

Graceful shutdown for the http server is also enabled and can be configured through props.

To configure server, MySQL, Redis, Consul and so on, see PROPS_* in prop.go.

It's also possible to register callbacks that are triggered before/after server bootstrap

	miso.PreServerBootstrap(func(c Rail) error {
		// do something right after configuration being loaded, but server hasn't been bootstraped yet
	});

	miso.PostServerBootstrapped(func(c Rail) error {
		// do something after the server bootstrap
	});

	// start the server
	miso.BootstrapServer(os.Args)
*/
func BootstrapServer(args []string) {
	App().Bootstrap(args)
}

// check if the server is shutting down
func IsShuttingDown() bool {
	return App().IsShuttingDown()
}

// mark that the server is shutting down
func MarkServerShuttingDown() {
	App().MarkServerShuttingDown()
}

// Shutdown server
func Shutdown() {
	App().Shutdown()
}

// Add listener that is invoked when server is finally bootstrapped
//
// This usually means all server components are started, such as MySQL connection, Redis Connection and so on.
//
// Caller is free to call PostServerBootstrapped inside another PostServerBootstrapped callback.
func PostServerBootstrapped(callback func(rail Rail) error) {
	App().PostServerBootstrapped(callback)
}

// Add listener that is invoked before the server is fully bootstrapped
//
// This usually means that the configuration is loaded, and the logging is configured, but the server components are not yet initialized.
//
// Caller is free to call PostServerBootstrapped or PreServerBootstrap inside another PreServerBootstrap callback.
func PreServerBootstrap(callback func(rail Rail) error) {
	App().PreServerBootstrap(callback)

}

// Register server component bootstrap callback
//
// When such callback is invoked, configuration should be fully loaded, the callback is free to read the loaded configuration
// and decide whether or not the server component should be initialized, e.g., by checking if the enable flag is true.
func RegisterBootstrapCallback(bootstrapComponent ComponentBootstrap) {
	App().RegisterBootstrapCallback(bootstrapComponent)
}

// Register shutdown hook, hook should never panic
func AddShutdownHook(hook func()) {
	App().AddOrderedShutdownHook(DefShutdownOrder, hook)
}

func AddOrderedShutdownHook(order int, hook func()) {
	App().AddOrderedShutdownHook(order, hook)
}

type OrderedShutdownHook struct {
	Hook  func()
	Order int
}
