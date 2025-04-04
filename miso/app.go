package miso

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
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

type ComponentBootstrap struct {
	// name of the component.
	Name string
	// the actual bootstrap function.
	Bootstrap func(rail Rail) error
	// check whether component should be bootstraped
	Condition func(rail Rail) (bool, error)
	// order of which the components are bootstraped, natural order, it's by default 0.
	Order int
}

type MisoApp struct {
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

	store  *appStore
	config *AppConfig
}

// Get global miso app.
//
// Only one MisoApp is supported, this func always returns the same app.
func App() *MisoApp {
	return globalApp
}

// only one MisoApp is supported for now.
func newApp() *MisoApp {
	return &MisoApp{
		manualSigQuit: make(chan int, 15), // increase size to 15 to avoid blocking multiple Shutdown() calls
		configLoaded:  false,
		shuttingDown:  false,
		store:         &appStore{store: util.NewRWMap[string, any]()},
		config:        newAppConfig(),
	}
}

func (a *MisoApp) Config() *AppConfig {
	return a.config
}

func (a *MisoApp) Store() *appStore {
	return a.store
}

// Bootstrap miso app.
func (a *MisoApp) Bootstrap(args []string) {
	a.LoadConfig(args)

	osSigQuit := make(chan os.Signal, 2)
	signal.Notify(osSigQuit, os.Interrupt, syscall.SIGTERM)

	a.AddOrderedShutdownHook(0, a.markServerShuttingDown) // the first hook to be called
	rail := EmptyRail()

	start := time.Now().UnixMilli()
	defer a.triggerShutdownHook()

	appName := GetPropStr(PropAppName)
	if appName == "" {
		rail.Fatalf("Property '%s' is required", PropAppName)
	}

	rail.Infof("\n\n---------------------------------------------- starting %s -------------------------------------------------------\n", appName)
	rail.Infof("Miso Version: %s", version.Version)
	rail.Infof("Production Mode: %v", a.Config().GetPropBool(PropProdMode))

	// invoke callbacks to setup server, sometime we need to setup stuff right after the configuration being loaded
	{
		rail.Infof("Triggering PreServerBootstrap")
		start := time.Now()
		if e := a.callPreServerBootstrapListeners(rail); e != nil {
			rail.Errorf("Error occurred while trigger PreServerBootstrap callbacks, %v", e)
			return
		}
		rail.Infof("PreServerBootstrap finished, took: %v", time.Since(start))
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

		rail.Debugf("Starting to bootstrap component %-30s", sbc.Name)
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
	{
		rail.Infof("Triggering PostServerBootstrap")
		start := time.Now()
		if e := a.callPostServerBootstrapListeners(rail); e != nil {
			rail.Errorf("Error occurred while triggering PostServerBootstrap callbacks, %v", e)
			return
		}
		rail.Infof("PostServerBootstrap finished, took: %v", time.Since(start))
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
	a.Config().DefaultReadConfig(args)

	if err := a.configureLogging(); err != nil {
		panic(fmt.Errorf("configure logging failed, %v", err))
	}
	a.configLoaded = true
}

// Trigger shutdown hook
func (a *MisoApp) triggerShutdownHook() {
	timeout := a.Config().GetPropInt(PropServerGracefulShutdownTimeSec)

	f := util.RunAsync(func() (any, error) {
		a.shmu.Lock()
		defer a.shmu.Unlock()

		sort.Slice(a.shutdownHook, func(i, j int) bool { return a.shutdownHook[i].Order < a.shutdownHook[j].Order })
		for _, hook := range a.shutdownHook {
			util.PanicSafeFunc(hook.Hook)() // hook can never panic
		}
		return nil, nil
	})
	if timeout > 0 {
		timeoutDur := time.Duration(timeout * int(time.Second))
		_, err := f.TimedGet(int(timeoutDur / time.Millisecond))
		if err != nil {
			if errors.Is(err, util.ErrGetTimeout) {
				Warnf("Exceeded server graceful shutdown period (%v), stop waiting for shutdown hook execution", timeoutDur)
			} else {
				Errorf("Unexpected error occurred while executing shutdown hooks, %v", err)
			}
		}
	} else {
		_, err := f.Get()
		if err != nil {
			Errorf("Unexpected error occurred while executing shutdown hooks, %v", err)
		}
	}

}

// Register shutdown hook, hook should never panic
func (a *MisoApp) AddShutdownHook(hook func()) {
	caller := getCallerFnUpOne()
	a.AddOrderedShutdownHook(DefShutdownOrder, func() {
		Debugf("Triggering ShutdownHook: %v", caller)
		defer Debugf("ShutdownHook exited: %v", caller)
		hook()
	})
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
func (a *MisoApp) markServerShuttingDown() {
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

func (a *MisoApp) RegisterBootstrapCallback(bootstrapComponent ComponentBootstrap) {
	a.serverBootrapCallbacks = append(a.serverBootrapCallbacks, bootstrapComponent)
}

func (a *MisoApp) PostServerBootstrap(callback ...func(rail Rail) error) {
	if callback == nil {
		return
	}
	a.postServerBootstrapListener = append(a.postServerBootstrapListener, callback...)
}

func (a *MisoApp) PreServerBootstrap(callback ...func(rail Rail) error) {
	if callback == nil {
		return
	}
	a.preServerBootstrapListener = append(a.preServerBootstrapListener, callback...)
}

func (a *MisoApp) configureLogging() error {
	util.PanicLog = Errorf
	util.DebugLog = Debugf
	util.CliErrLog = Errorf
	c := a.Config()

	// determine the writer that we will use for logging (loggerOut and loggerErrOut)
	if c.HasProp(PropLoggingRollingFile) {
		logFile := c.GetPropStr(PropLoggingRollingFile)
		log := BuildRollingLogFileWriter(NewRollingLogFileParam{
			Filename:   logFile,
			MaxSize:    c.GetPropInt(PropLoggingRollingFileMaxSize), // megabytes
			MaxAge:     c.GetPropInt(PropLoggingRollingFileMaxAge),  //days
			MaxBackups: c.GetPropInt(PropLoggingRollingFileMaxBackups),
		})
		loggerOut = log
		loggerErrOut = log

		if c.GetPropBool(PropLoggingRollingFileRotateDaily) {
			// schedule a job to rotate the log at 00:00:00
			if err := ScheduleCron(Job{
				Name:            "RotateLogJob",
				Cron:            "0 0 0 * * ?",
				CronWithSeconds: true,
				Run:             func(r Rail) error { return log.Rotate() },
			}); err != nil {
				return fmt.Errorf("failed to register RotateLogJob, %v", err)
			}
		}
	}

	SetLogOutput(loggerOut)

	if c.HasProp(PropLoggingLevel) {
		SetLogLevel(c.GetPropStr(PropLoggingLevel))
	}
	return nil
}

// Configure logging level and output target based on loaded configuration.
func ConfigureLogging() error {
	return App().configureLogging()
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

	miso.PostServerBootstrap(func(c Rail) error {
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

// Shutdown server
func Shutdown() {
	App().Shutdown()
}

// deprecated: use PostServerBootstrap(...) instead.
var PostServerBootstrapped = PostServerBootstrap

// Add listener that is invoked when server is finally bootstrapped
//
// This usually means all server components are started, such as MySQL connection, Redis Connection and so on.
//
// Caller is free to call PostServerBootstrap inside another PostServerBootstrap callback.
func PostServerBootstrap(f ...func(rail Rail) error) {
	App().PostServerBootstrap(f...)
}

// Add listener that is invoked before the server is fully bootstrapped
//
// This usually means that the configuration is loaded, and the logging is configured, but the server components are not yet initialized.
//
// Caller is free to call PostServerBootstrap or PreServerBootstrap inside another PreServerBootstrap callback.
func PreServerBootstrap(f ...func(rail Rail) error) {
	App().PreServerBootstrap(f...)
}

// Register server component bootstrap callback
//
// When such callback is invoked, configuration should be fully loaded, the callback is free to read the loaded configuration
// and decide whether or not the server component should be initialized, e.g., by checking if the enable flag is true.
func RegisterBootstrapCallback(c ComponentBootstrap) {
	App().RegisterBootstrapCallback(c)
}

// Register shutdown hook, hook should never panic
func AddShutdownHook(hook func()) {
	App().AddShutdownHook(hook)
}

func AddOrderedShutdownHook(order int, hook func()) {
	App().AddOrderedShutdownHook(order, hook)
}

type OrderedShutdownHook struct {
	Hook  func()
	Order int
}

type appStore struct {
	store *util.RWMap[string, any]
}

func (a *appStore) Get(k string) (any, bool) {
	return a.store.Get(k)
}

func (a *appStore) GetElse(k string, elseFunc func(k string) any) (any, bool) {
	return a.store.GetElse(k, elseFunc)
}

func (a *appStore) Put(k string, v any) {
	a.store.Put(k, v)
}

func (a *appStore) Del(k string) {
	a.store.Del(k)
}

func AppStoreGet[V any](app *MisoApp, k string) V {
	var vt V
	v, ok := app.Store().Get(k)
	if !ok {
		return vt
	}
	if v != nil {
		return v.(V)
	}
	return vt
}

func AppStoreGetElse[V any](app *MisoApp, k string, f func() V) V {
	var vt V
	v, _ := app.Store().GetElse(k, func(k string) any { return f() })
	if v != nil {
		return v.(V)
	}
	return vt
}

func InitAppModuleFunc[V any](initFunc func() V) func() V {
	t := reflect.TypeOf(util.NewVar[V]())
	k := util.TypeName(t)
	if k == "" {
		panic(fmt.Errorf("cannot obtain type name of %v, unable to create InitAppModuleFunc", t))
	}
	appModule := func() V {
		app := App()
		return AppStoreGetElse[V](app, k, func() V {
			return initFunc()
		})
	}
	return appModule
}
