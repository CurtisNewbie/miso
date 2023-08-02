#!/bin/bash

go version
if [[ $? -ne 0 ]]; then
    return 1
fi

file="./app-conf-dev.yml"
if [[ -z "$file" ]]; then
    touch "$file"
fi

echo "app.name: 'demo'" > "$file"
echo "mode.production: false # enable production mode" >> "$file"
echo "" >> "$file"

echo "mysql:" >> "$file"
echo "  enabled: false" >> "$file"
echo "  user:" >> "$file"
echo "  password:" >> "$file"
echo "  database:" >> "$file"
echo "  host: localhost" >> "$file"
echo "  port: 3306" >> "$file"
echo "" >> "$file"

echo "redis:" >> "$file"
echo "  enabled: false" >> "$file"
echo "  address: localhost" >> "$file"
echo "  port: 6379" >> "$file"
echo "  password:" >> "$file"
echo "  database: 0" >> "$file"
echo "" >> "$file"

echo "task:" >> "$file"
echo "  scheduling: " >> "$file"
echo "    group: '\${app.name}'" >> "$file"
echo "    enabled: true" >> "$file"
echo "" >> "$file"

echo "server:" >> "$file"
echo "  enabled: true" >> "$file"
echo "  host: localhost" >> "$file"
echo "  port: 8080" >> "$file"
echo "  gracefulShutdownTimeSec: 5" >> "$file"
echo "  perf.enabled: false" >> "$file"
echo "" >> "$file"

echo "consul:" >> "$file"
echo "  enabled: false" >> "$file"
echo "  registerName: '\${app.name}'" >> "$file"
echo "  consulAddress: localhost:8500" >> "$file"
echo "  healthCheckUrl: /health" >> "$file"
echo "  healthCheckInterval: 15s" >> "$file"
echo "  healthCheckTimeout: 3s" >> "$file"
echo "  healthCheckFailedDeregisterAfter: 120s" >> "$file"
echo "" >> "$file"

echo "logging:" >> "$file"
echo "#  rolling.file: '\${app.name}.log'" >> "$file"
echo "  level: 'info'" >> "$file"
echo "" >> "$file"

echo "rabbitmq:" >> "$file"
echo "  enabled: false" >> "$file"
echo "  host: localhost" >> "$file"
echo "  port: 5672" >> "$file"
echo "  consumer.qos: 68" >> "$file"
echo "  username: ''" >> "$file"
echo "  password: ''" >> "$file"
echo "  vhost: ''" >> "$file"
echo "" >> "$file"

echo "jwt:" >> "$file"
echo "  key:" >> "$file"
echo "    public: ''" >> "$file"
echo "    private: ''" >> "$file"
echo "    issuer: ''" >> "$file"
echo "" >> "$file"

echo "# tracing.propagation.keys:" >> "$file"
echo "#  - " >> "$file"
echo "#  - " >> "$file"


file="./main.go"
if [[ -z "$file" ]]; then
    touch "$file"
else
    echo "" > "$file"
fi
 >> "$file"
echo "package main" >> "$file"
echo "" >> "$file"
echo "import (" >> "$file"
echo "    \"os\"" >> "$file"
echo "    \"github.com/curtisnewbie/gocommon/server\"" >> "$file"
echo ")" >> "$file"
echo "" >> "$file"
echo "func main() {" >> "$file"
echo "    server.BootstrapServer(os.Args)" >> "$file"
echo "}" >> "$file"
echo "" >> "$file"

go mod init demo && \
    go get github.com/curtisnewbie/gocommon@HEAD && \
    go mod tidy

echo "Project initialized at $(pwd)"