# Application Lifecycle

Miso provides a few lifecycle callbacks for user to hook callbacks into it. Before any hooks are triggered, Miso must load the configuration first.

Callbacks registered by `server.PreServerBootstrap(...)` are invoked right after Miso loaded configuration from ENV, CLI args and configuration files. From this point, Miso hasn't yet started boostraping.

After all `PreServerBoostrap` callbacks are invoked. Miso then starts boostraping server components by invoking the callbacks registered using `server.RegisterBootstrapCallback(...)`. The initialization for builtin components like MySQL clients, are just handled extactly the same way like this.

After all `RegisterBootstrapCallback` callbacks are invoked, Miso assumes that the server is fully bootstrapped, it then starts invoking callbacks regsitered using `server.PostServerBootstrapped(...)`.