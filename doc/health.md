# HealthCheck

Services' health are periodically check by Consul. You can register a health indicator for your service component as follows:

```go
	AddHealthIndicator(HealthIndicator{
		Name: "My Component",
		CheckHealth: func(rail Rail) bool {
            err := checkHealth()
			if err != nil {
				rail.Errorf("my component is down, %v", err)
				return false
			}
			return true
		},
	})
```

Everytime the healthcheck endpoint is called, all registered `HealthIndicator`s are called. If any `HealthIndicator` is DOWN, the healthcheck endpoint returns 503.