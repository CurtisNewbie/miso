# Application Lifecycle

Miso provides a few lifecycle callbacks. Before any callbacks are triggered, Miso must load the configuration first.

Callbacks registered by `miso.PreServerBootstrap(...)` are invoked right after Miso loaded configuration from ENV, CLI args and configuration files. From this point, Miso hasn't yet started boostraping.

After all `PreServerBoostrap` callbacks are invoked. Miso then starts boostraping server components by invoking the callbacks registered using `miso.RegisterBootstrapCallback(...)`. The initialization for builtin components like MySQL clients, are just handled extactly the same way like this.

After all `RegisterBootstrapCallback` callbacks are invoked, Miso assumes that the server is fully bootstrapped, it then starts invoking callbacks regsitered using `miso.PostServerBootstrap(...)`.