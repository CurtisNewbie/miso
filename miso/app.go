package miso

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/rfutil"
	"github.com/curtisnewbie/miso/util/utillog"
	"github.com/curtisnewbie/miso/version"
	"github.com/google/gops/agent"
	"go.uber.org/automaxprocs/maxprocs"
)

const (
	// Default shutdown hook execution order.
	DefShutdownOrder = 5
)

const (
	// Components like database that are essential and must be ready before anything else.
	//
	// Since v0.3.2, L1 was updated from -20 to -30.
	BootstrapOrderL1 = -30

	// Components that are bootstraped before the web server, such as metrics stuff.
	//
	// Since v0.3.2, L2 was updated from -15 to -20.
	BootstrapOrderL2 = -20

	// The web server or anything similar, bootstraping web server doesn't really mean that we will receive inbound requests.
	BootstrapOrderL3 = -10

	// Default bootstrap order 0.
	BootstrapOrderDefault = 0

	// Components that introduce inbound requests or job scheduling.
	//
	// When these components bootstrap, the server is considered truly running.
	// For example, service registration (for service discovery), MQ broker connection and so on.
	//
	// Since v0.3.2, L4 was updated from -5 to 10.
	BootstrapOrderL4 = 10
)

const (
	misoAppHealthIndicatorName = "MisoAppBoostrap"
)

var (
	loggerOut    io.Writer = os.Stdout
	loggerErrOut io.Writer = os.Stderr

	globalApp *MisoApp = newApp()
)

func init() {
	maxprocs.Set()
}

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
	configLoaded     bool
	fullyBoostrapped *atomic.Bool

	// channel for signaling server shutdown
	manualSigQuit chan int

	shuttingDown *atomic.Bool
	shutdownHook []OrderedShutdownHook
	shmu         sync.Mutex // mutex for shutdownHook

	serverBootrapCallbacks      []ComponentBootstrap
	configLoader                []func(r Rail) error
	preServerBootstrapListener  []func(r Rail) error
	postServerBootstrapListener []func(r Rail) error
	appReadyListener            []func(r Rail) error

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
	a := &MisoApp{
		manualSigQuit:    make(chan int, 15), // increase size to 15 to avoid blocking multiple Shutdown() calls
		configLoaded:     false,
		shuttingDown:     &atomic.Bool{},
		store:            &appStore{store: hash.NewStrRWMap[any]()},
		config:           newAppConfig(),
		fullyBoostrapped: &atomic.Bool{},
	}
	return a
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
	a.changeLogLevel()

	osSigQuit := make(chan os.Signal, 2)
	signal.Notify(osSigQuit, os.Interrupt, syscall.SIGTERM)

	rail := EmptyRail()
	start := time.Now().UnixMilli()
	defer a.triggerShutdownHook()
	defer a.markServerShuttingDown()

	appName := GetPropStr(PropAppName)
	if appName == "" {
		rail.Fatalf("Property '%s' is required", PropAppName)
	}

	if e := a.callConfigLoaders(rail); e != nil {
		rail.Errorf("Error occurred while running ConfigLoader, %v", e)
		return
	}

	if err := a.configureLogging(); err != nil {
		rail.Errorf("Configure logging failed, %v", err)
		return
	}

	split := strings.Repeat("-", 58)
	rail.Infof("\n\n%s starting %s %s\n", split, appName, split)
	rail.Infof("Miso Version: %s", version.Version)
	rail.Infof("Production Mode: %v", a.Config().GetPropBool(PropProdMode))
	rail.Infof("CPUs: %v, GOMAXPROCS: %v", runtime.NumCPU(), runtime.GOMAXPROCS(0))

	// initiate gops
	if err := agent.Listen(agent.Options{}); err != nil {
		rail.Errorf("Failed to create gops agent, %v", err)
		return
	}
	AddShutdownHook(func() { agent.Close() })
	rail.Debug("Created gops agent")

	// bootstrap health indicator
	a.addBootstrapHealthIndicator()

	// invoke callbacks to setup server, sometime we need to setup stuff right after the configuration being loaded
	if e := a.callPreServerBootstrapListeners(rail); e != nil {
		rail.Errorf("Error occurred while trigger PreServerBootstrap callbacks, %v", e)
		return
	}

	// bootstrap components, these are sorted by their orders
	if err := a.callBoostrapComponents(rail); err != nil {
		rail.Errorf("Boostrap server components failed, %v", err)
		return
	}

	// invoke listener for serverBootstraped event
	if e := a.callPostServerBootstrapListeners(rail); e != nil {
		rail.Errorf("Error occurred while triggering PostServerBootstrap callbacks, %v", e)
		return
	}

	// marked as fully bootstrapped
	a.fullyBoostrapped.Store(true)

	// invoke listener for appReady event
	{
		rail.Infof("Triggering OnAppReady")
		start := time.Now()
		if e := a.callAppReadyListeners(rail); e != nil {
			rail.Errorf("Error occurred while triggering OnAppReady callbacks, %v", e)
			return
		}
		rail.Infof("OnAppReady finished, took: %v", time.Since(start))
	}

	end := time.Now().UnixMilli()
	split = strings.Repeat("-", 52)
	rail.Infof("\n\n%s %s started (took: %dms) %s\n", split, appName, end-start, split)

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
	a.configLoaded = true
}

func (a *MisoApp) changeLogLevel() {
	if a.config.HasProp(PropLoggingLevel) {
		SetLogLevel(a.config.GetPropStr(PropLoggingLevel))
	}
}

func (a *MisoApp) addBootstrapHealthIndicator() {
	AddHealthIndicator(HealthIndicator{
		Name: misoAppHealthIndicatorName,
		CheckHealth: func(rail Rail) bool {
			return a.fullyBoostrapped.Load()
		},
	})
}

// Trigger shutdown hook
func (a *MisoApp) triggerShutdownHook() {
	Info("Triggering shutdown hooks")
	timeout := a.Config().GetPropInt(PropServerGracefulShutdownTimeSec)
	panicSafeFunc := func(op func() util.Future[any]) func() util.Future[any] {
		return func() util.Future[any] {
			defer func() {
				if v := recover(); v != nil {
					Errorf("panic recovered, %v\n%v", v, util.UnsafeByt2Str(debug.Stack()))
				}
			}()
			return op()
		}
	}

	f := util.RunAsync(func() (any, error) {
		a.shmu.Lock()
		defer a.shmu.Unlock()

		sort.Slice(a.shutdownHook, func(i, j int) bool { return a.shutdownHook[i].Order < a.shutdownHook[j].Order })
		futures := make([]util.Future[any], 0, len(a.shutdownHook))
		for _, hook := range a.shutdownHook {
			futures = append(futures, panicSafeFunc(hook.Hook)())
		}
		for _, hookFtr := range futures {
			_, _ = hookFtr.Get()
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

func (a *MisoApp) AddShutdownHook(hook func()) {
	caller := getCallerFnUpN(1)
	a.AddOrderedShutdownHook(DefShutdownOrder, func() {
		start := time.Now()
		Infof("Triggering ShutdownHook: %v", caller)
		defer func() { Infof("ShutdownHook exited: %v, took: %v", caller, time.Since(start)) }()
		hook()
	})
}

func (a *MisoApp) AddAsyncShutdownHook(hook func()) {
	caller := getCallerFnUpN(1)
	a.AddOrderedAsyncShutdownHook(DefShutdownOrder, func() {
		start := time.Now()
		Infof("Triggering Async ShutdownHook: %v", caller)
		defer func() { Infof("Async ShutdownHook exited: %v, took: %v", caller, time.Since(start)) }()
		hook()
	})
}

func (a *MisoApp) AddOrderedAsyncShutdownHook(order int, hook func()) {
	a.shmu.Lock()
	defer a.shmu.Unlock()
	a.shutdownHook = append(a.shutdownHook, OrderedShutdownHook{
		Order: order,
		Hook: func() util.Future[any] {
			return util.RunAsync(func() (any, error) {
				hook()
				return nil, nil
			})
		},
	})
}

func (a *MisoApp) AddOrderedShutdownHook(order int, hook func()) {
	a.shmu.Lock()
	defer a.shmu.Unlock()
	a.shutdownHook = append(a.shutdownHook, OrderedShutdownHook{
		Order: order,
		Hook: func() util.Future[any] {
			hook()
			return util.NewCompletedFuture[any](nil, nil)
		},
	})
}

// check if the server is shutting down
func (a *MisoApp) IsShuttingDown() bool {
	return a.shuttingDown.Load()
}

// mark that the server is shutting down
func (a *MisoApp) markServerShuttingDown() {
	a.shuttingDown.Store(true)
}

// Shutdown server
func (a *MisoApp) Shutdown() {
	a.manualSigQuit <- 1
}

func (a *MisoApp) callAppReadyListeners(rail Rail) error {
	for _, c := range a.appReadyListener {
		if e := c(rail); e != nil {
			return e
		}
	}
	a.appReadyListener = nil
	return nil
}

func (a *MisoApp) callPostServerBootstrapListeners(rail Rail) error {
	rail.Infof("Triggering PostServerBootstrap")
	start := time.Now()

	i := 0
	for i < len(a.postServerBootstrapListener) {
		if e := a.postServerBootstrapListener[i](rail); e != nil {
			return e
		}
		i++
	}
	a.postServerBootstrapListener = nil
	rail.Infof("PostServerBootstrap finished, took: %v", time.Since(start))
	return nil
}

func (a *MisoApp) callConfigLoaders(rail Rail) error {
	if len(a.configLoader) < 1 {
		return nil
	}
	rail.Infof("Running ConfigLoader")
	start := time.Now()

	i := 0
	for i < len(a.configLoader) {
		if e := a.configLoader[i](rail); e != nil {
			return e
		}
		i++
	}
	a.configLoader = nil
	a.changeLogLevel()
	rail.Infof("ConfigLoader finished, took: %v", time.Since(start))
	return nil
}

func (a *MisoApp) callPreServerBootstrapListeners(rail Rail) error {
	rail.Infof("Triggering PreServerBootstrap")
	start := time.Now()

	i := 0
	for i < len(a.preServerBootstrapListener) {
		if e := a.preServerBootstrapListener[i](rail); e != nil {
			return e
		}
		i++
	}
	a.preServerBootstrapListener = nil
	rail.Infof("PreServerBootstrap finished, took: %v", time.Since(start))
	return nil
}

func (a *MisoApp) callBoostrapComponents(rail Rail) error {
	sort.Slice(a.serverBootrapCallbacks, func(i, j int) bool { return a.serverBootrapCallbacks[i].Order < a.serverBootrapCallbacks[j].Order })
	Debugf("serverBootrapCallbacks: %+v", a.serverBootrapCallbacks)
	slowBootstrapThreshold := GetPropDuration(PropAppSlowBoostrapThresohold)
	for _, sbc := range a.serverBootrapCallbacks {
		if sbc.Condition != nil {
			ok, ce := sbc.Condition(rail)
			if ce != nil {
				return errs.WrapErrf(ce, "failed to bootstrap server component: %v, failed on condition check", sbc.Name)
			}
			if !ok {
				continue
			}
		}

		rail.Debugf("Starting to bootstrap component %-30s", sbc.Name)
		start := time.Now()
		if e := sbc.Bootstrap(rail); e != nil {
			return errs.WrapErrf(e, "failed to bootstrap server component: %v", sbc.Name)
		}
		took := time.Since(start)
		rail.Debugf("Callback %-30s - took %v", sbc.Name, took)
		if took >= slowBootstrapThreshold {
			rail.Warnf("Component '%s' might be too slow to bootstrap, took: %v", sbc.Name, took)
		}
	}
	a.serverBootrapCallbacks = nil
	return nil
}

func (a *MisoApp) RegisterBootstrapCallback(bootstrapComponent ComponentBootstrap) {
	a.serverBootrapCallbacks = append(a.serverBootrapCallbacks, bootstrapComponent)
}

func (a *MisoApp) OnAppReady(callback ...func(rail Rail) error) {
	if callback == nil {
		return
	}
	a.appReadyListener = append(a.appReadyListener, callback...)
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

func (a *MisoApp) RegisterConfigLoader(callback ...func(rail Rail) error) {
	if callback == nil {
		return
	}
	a.configLoader = append(a.configLoader, callback...)
}

func (a *MisoApp) configureLogging() error {
	utillog.ErrorLog = Errorf
	utillog.DebugLog = Debugf
	c := a.Config()

	// determine the writer that we will use for logging (loggerOut and loggerErrOut)
	if c.HasProp(PropLoggingRollingFile) {
		logFile := c.GetPropStr(PropLoggingRollingFile)

		if logFile != "" && c.GetPropBool(PropLoggingRollingFileAppendIpSuffix) {
			n, ok := util.FileCutSuffix(logFile, "log")
			if ok {
				logFile = n + "-" + util.GetLocalIPV4() + ".log"
			}
		}

		log := BuildRollingLogFileWriter(NewRollingLogFileParam{
			Filename:   logFile,
			MaxSize:    c.GetPropInt(PropLoggingRollingFileMaxSize), // megabytes
			MaxAge:     c.GetPropInt(PropLoggingRollingFileMaxAge),  //days
			MaxBackups: c.GetPropInt(PropLoggingRollingFileMaxBackups),
		})

		if c.GetPropBool(PropLoggingRollingFileOnly) {
			loggerOut = log
			loggerErrOut = log
		} else {
			loggerOut = io.MultiWriter(os.Stdout, log)
			loggerErrOut = io.MultiWriter(os.Stderr, log)
		}

		if c.GetPropBool(PropLoggingRollingFileRotateDaily) {
			if !IsProdMode() { // rotate immediately in dev mode
				log.Rotate()
			}

			// schedule a job to rotate the log at 00:00:00
			if err := ScheduleCron(Job{
				Name: "RotateLogJob",
				Cron: "0 0 0 * * ?",
				Run:  func(r Rail) error { return log.Rotate() },
			}); err != nil {
				return fmt.Errorf("failed to register RotateLogJob, %v", err)
			}
		}
		Infof("Configured Log File: %v", logFile)
	}

	SetLogOutput(loggerOut)

	a.changeLogLevel()

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

// Add listener that is invoked when server is ready.
//
// OnAppReady(...) callbacks are invoked after PostServerBoostrap().
func OnAppReady(f ...func(rail Rail) error) {
	App().OnAppReady(f...)
}

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

func RegisterConfigLoader(callback ...func(rail Rail) error) {
	App().RegisterConfigLoader(callback...)
}

// Register server component bootstrap callback
//
// When such callback is invoked, configuration should be fully loaded, the callback is free to read the loaded configuration
// and decide whether or not the server component should be initialized, e.g., by checking if the enable flag is true.
func RegisterBootstrapCallback(c ComponentBootstrap) {
	App().RegisterBootstrapCallback(c)
}

func AddShutdownHook(hook func()) {
	App().AddShutdownHook(hook)
}

func AddAsyncShutdownHook(hook func()) {
	App().AddAsyncShutdownHook(hook)
}

func AddOrderedShutdownHook(order int, hook func()) {
	App().AddOrderedShutdownHook(order, hook)
}

func AddOrderedAsyncShutdownHook(order int, hook func()) {
	App().AddOrderedAsyncShutdownHook(order, hook)
}

type OrderedShutdownHook struct {
	Hook  func() util.Future[any]
	Order int
}

type appStore struct {
	store *hash.StrRWMap[any]
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
	t := reflect.TypeOf(rfutil.NewVar[V]())
	k := rfutil.TypeName(t)
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
