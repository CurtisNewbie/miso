# Utility Middleware

Standalone middleware packages for common utility needs: crypto, jwt, expr, lua, money, zk.

## Crypto

AES-ECB and RSA encryption, plus hash helpers.

**Package:** `github.com/curtisnewbie/miso/middleware/crypto`

```go
// AES-ECB encrypt/decrypt (PKCS padding), returns base64 string
enc, err := crypto.AesEcbEncrypt([]byte("secret-key-16-byte"), "plain text")
dec, err := crypto.AesEcbDecrypt([]byte("secret-key-16-byte"), enc)

// Hashing
md5Hex  := crypto.MD5Hex([]byte("data"))      // []byte → hex
md5Str  := crypto.MD5HexStr("data")           // string → hex
sha1    := crypto.SHA1HexStr("data")
sha256  := crypto.SHA256HexStr("data")
sha512  := crypto.SHA512HexStr("data")

// RSA keys from PEM content (PKCS1/PKCS8 private key, PKIX public key)
priv, err := crypto.LoadPrivKey(pemContent)
pub, err  := crypto.LoadPubKey(pemContent)
```

## JWT

RS256 JWT encode/decode. Keys come from config props, or pass keys explicitly.

**Package:** `github.com/curtisnewbie/miso/middleware/jwt`

```yaml
jwt:
  key:
    public: "-----BEGIN PUBLIC KEY-----..."   # jwt.key.public
    private: "-----BEGIN PRIVATE KEY-----..." # jwt.key.private
    issuer: "myapp"                            # jwt.key.issuer
```

```go
import "github.com/curtisnewbie/miso/middleware/jwt"
import "github.com/golang-jwt/jwt/v5"

// Encode/decode using configured keys and issuer
token, err := jwt.JwtEncode(jwt.MapClaims{"uid": "1"}, 24*time.Hour)
parsed, err := jwt.JwtDecode(token)
if parsed.Valid {
    uid := parsed.Claims["uid"]
}

// Encode/decode with explicit key and issuer (ignores props)
token, err = jwt.JwtKeyEncode(privKey, jwt.MapClaims{"uid": "1"}, 24*time.Hour, "myapp")
parsed, err = jwt.JwtKeyDecode(pubKey, token, "myapp")
```

## Expr

Compile and evaluate [expr-lang](https://expr-lang.org) expressions.

**Package:** `github.com/curtisnewbie/miso/middleware/expr`

```go
import "github.com/curtisnewbie/miso/middleware/expr"

// Compile once, evaluate many times (typed environment)
compiled, err := expr.Compile[User]("age > 18 && vip")
ok, err := compiled.Eval(User{Age: 20, Vip: true})

// With inline environment
compiled, err := expr.CompileEnv("age > $threshold", Env{Threshold: 18})

// Eval with any environment type
v, err := expr.Eval("1 + 2", nil)

// Thread-safe LRU-compiled pool (caches compiled expressions)
pool := expr.NewPooledExpr[User](100)
v, err := pool.Eval("age > 18", User{Age: 20})
```

## Lua

Run Lua scripts (gopher-lua) with typed return values and builtin logging/globals.

**Package:** `github.com/curtisnewbie/miso/middleware/lua`

```go
import "github.com/curtisnewbie/miso/middleware/lua"

// T can be int, float64, string, bool
n, err := lua.Run[int](`return 1 + 2`, lua.WithLogger(rail))

s, err := lua.Run[string](`return "hello " .. name`, lua.WithGlobalStr("name", "world"))

// Builtin funcs inside scripts: printf(...), infof(...), errorf(...)
// (infof/errorf write to the Rail logger when WithLogger is provided, otherwise stdout)

// Run from file or reader
v, err := lua.RunFile[float64]("scripts/calc.lua", lua.WithGlobalNum("x", 3.14))
```

Options: `WithLogger(rail)`, `WithGlobalStr/WithGlobalNum/WithGlobalBool/WithGlobalNil/WithGlobalStrTable`.

## Money

Arbitrary-precision decimal amounts (wraps `inf.Dec`) for money math.

**Package:** `github.com/curtisnewbie/miso/middleware/money`

```go
import "github.com/curtisnewbie/miso/middleware/money"

a := money.NewAmt("19.99")  // from string (no float precision loss)
b := money.NewAmt("0.01")

sum := a.Add(b)
diff := a.Sub(b)
mul := a.Mul(b)
quot := a.Div(b, 2)            // scale 2
rounded := a.Round(2)

// Currency-aware rounding (uses minor-unit scale of the currency)
usd, err := a.RoundCurrency("USD")

cmp := a.Cmp(b)                // -1, 0, 1
zero := money.Zero()
```

- JSON marshals as **string** by default (`"19.99"`); call `money.AmtMarshalAsNum(true)` to marshal as number
- Implements `sql.Scanner`/`driver.Valuer`, safe in GORM
- Free functions also available: `money.Add/Sub/Mul/Div/Round/UnitDec/UnitFmt/Unit/Scale/UnitScale`

## ZK (ZooKeeper)

ZooKeeper connection management and leader election.

**Package:** `github.com/curtisnewbie/miso/middleware/zk`

```go
import "github.com/curtisnewbie/miso/middleware/zk"

conn := zk.Conn() // managed *zk.Conn

// Create ephemeral / persistent nodes
err := zk.CreateEphNode("/workers/node-1", []byte("data"))
err := zk.CreatePerNode("/config/app", []byte("{}"))

// Watch node changes
events, err := zk.Watch("/config/app")
for evt := range events {
    rail.Infof("Node changed: %v", evt.Type)
}

// Read node data
data, err := zk.Get("/config/app")

// Leader election — blocks until this node becomes leader, then runs leaderDo
election := zk.NewLeaderElection("/leader/app")
elected, err := election.Elect(rail, func() {
    // leader-only work (e.g., start cron scheduler)
})
```
