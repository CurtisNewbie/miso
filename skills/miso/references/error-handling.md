# Error Handling

Comprehensive error handling patterns using the `errs` package and Rail logging.

## Creating Errors

### Simple Error

```go
import "github.com/curtisnewbie/miso/errs"

err := errs.NewErrf("operation failed")
```

### Error with Code

```go
err := errs.NewErrfCode("USER_NOT_FOUND", "User does not exist")
```

### Define Error with Code Constant

```go
const (
    FileNotFound  = "FILE_NOT_FOUND"
    FileDeleted   = "FILE_DELETED"
)

var (
    ErrFileNotFound = errs.NewErrfCode(FileNotFound, "File is not found")
    ErrFileDeleted  = errs.NewErrfCode(FileDeleted, "File is deleted")
)

// Use the error
if !exists {
    return ErrFileNotFound.New()
}
```

**Why use `.New()`?**

`.New()` creates a new error instance with a fresh stack trace pointing to the actual error location. Without `.New()`, the stack trace would point to where the error variable was defined, making debugging difficult.

```go
var ErrFileNotFound = errs.NewErrfCode(FileNotFound, "File is not found")

// Direct return - stack trace points to ErrFileNotFound definition
return ErrFileNotFound  // ❌ Wrong stack trace location

// Using .New() - stack trace points to the actual error site
return ErrFileNotFound.New()  // ✅ Correct stack trace location
```

### Error with Internal Message

```go
err := errs.NewErrf("operation failed").
    WithInternalMsg("detailed debug information for developers")
```

### Error with Code and Internal Message

```go
err := errs.NewErrfCode("DB_ERROR", "Database operation failed").
    WithInternalMsg("query failed: SELECT * FROM users WHERE id = ?, err: connection timeout")
```

## Wrapping Errors

### Wrap Existing Error

```go
return errs.WrapErr(err, "failed to process request")
```

### Wrap with Formatted Message

```go
return errs.WrapErrf(err, "failed to load user: %s", userID)
```

### Wrap with Internal Message

```go
err := errs.WrapErr(dbErr, "database query failed").
    WithInternalMsg("query: SELECT * FROM users WHERE id = ?, args: [123]")
```

## Error Methods

```go
// Message returned to client
msg := err.Msg()

// Internal debug message (server-only)
internalMsg := err.InternalMsg()

// Error code string
code := err.Code()

// Stack trace
stackTrace := err.StackTrace()

// Convert to standard error
stdErr := err.Error()
```

## Checking Errors

### Check for None/Not Found Error

```go
if errs.IsNoneErr(err) {
    // Handle not found case
    return nil
}
```

### Check for Specific Errors

```go
if errs.IsAny(e, fstore.ErrFileNotFound, fstore.ErrFileDeleted) {
    return "", errs.NewErrf("File not found or deleted")
}
```

### Check for Standard Error Types

```go
import "errors"

if errors.Is(err, gorm.ErrRecordNotFound) {
    return errs.NewErrfCode("USER_NOT_FOUND", "User does not exist")
}
```

## Error Response Format

Framework automatically converts `MisoErr` to JSON responses:

```json
{
  "errorCode": "USER_NOT_FOUND",
  "msg": "User does not exist",
  "error": true,
  "data": null
}
```

### Success Response

```json
{
  "errorCode": "",
  "msg": "ok",
  "error": false,
  "data": {
    "userId": "123",
    "name": "John"
  }
}
```

## Validation

miso provides built-in struct validation using the `valid` tag (preferred) or `validation` tag (deprecated).

### Basic Usage

```go
type CreateUserReq struct {
    Name  string `json:"name" valid:"notEmpty"`
    Email string `json:"email" valid:"notEmpty"`
    Age   int    `json:"age" valid:"positive"`
}

func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error) {
    if err := miso.Validate(req); err != nil {
        if ve, ok := err.(*miso.ValidationError); ok {
            return errs.WrapErrf(err, "validation failed: %s", ve.Error())
        }
        return errs.WrapErr(err, "validation failed")
    }
    // ...
}
```

### Available Validation Rules

| Rule | Description | Example |
|------|-------------|---------|
| `notEmpty` | Non-empty string, array, slice, or map | `valid:"notEmpty"` or `valid:"notEmpty:Name is required"` |
| `notNil` | Non-nil pointer, slice, map, or func | `valid:"notNil"` |
| `maxLen` | Maximum length of string, array, or slice | `valid:"maxLen:100"` |
| `positive` | Greater than zero (int or string) | `valid:"positive"` |
| `positiveOrZero` | Greater than or equal to zero | `valid:"positiveOrZero"` |
| `negative` | Less than zero | `valid:"negative"` |
| `negativeOrZero` | Less than or equal to zero | `valid:"negativeOrZero"` |
| `notZero` | Not equal to zero | `valid:"notZero"` |
| `member` | Must be one of the specified values (string only) | `valid:"member:PUBLIC\|PROTECTED\|PRIVATE"` |
| `validated` | Validate nested struct/slice recursively | `valid:"validated"` |
| `trim` | Automatically trim string values before validation | `valid:"trim"` |

### Multiple Rules

Combine multiple rules with commas:

```go
type User struct {
    Name  string `valid:"trim,notEmpty"`
    Email string `valid:"trim,notEmpty,maxLen:255"`
    Age   int    `valid:"positiveOrZero"`
    Role  string `valid:"member:ADMIN\|USER\|GUEST"`
}

type CreateUserReq struct {
    User  *User  `valid:"notNil,validated"`
}
```

### Validation Error

When validation fails, a `ValidationError` is returned with:

```go
type ValidationError struct {
    Field               string // Field name
    Rule                string // Rule that failed
    ValidationMsg       string // Default validation message
    CustomValidationMsg string // Custom message (e.g., from `notEmpty:Custom message"`)
}

func (ve *ValidationError) Error() string {
    if ve.CustomValidationMsg != "" {
        return ve.CustomValidationMsg
    }
    return ve.Field + " " + ve.ValidationMsg
}
```

### Inline Validation in API Handlers

For automatic validation in API handlers, use the `valid` tag directly on request struct fields. miso's server validates inbound requests automatically when `server.validate.request.enabled=true` (default).

```go
type LoginReq struct {
    Username string `json:"username" valid:"trim,notEmpty"`
    Password string `json:"password" valid:"notEmpty"`
}

func Login(inb *miso.Inbound, req LoginReq) (LoginRes, error) {
    // No manual validation needed - miso validates automatically
    // If validation fails, an error is returned before this function runs
    // ...
}
```

## Error Code Conventions

Use descriptive error codes:

```go
const (
    ErrCodeSuccess            = "SUCCESS"
    ErrCodeValidationFailed   = "VALIDATION_ERROR"
    ErrCodeUnauthorized       = "UNAUTHORIZED"
    ErrCodeForbidden          = "PERMISSION_DENIED"
    ErrCodeNotFound           = "NOT_FOUND"
    ErrCodeConflict           = "CONFLICT"
    ErrCodeInternalError      = "INTERNAL_ERROR"
    ErrCodeExternalAPIError   = "EXTERNAL_API_ERROR"
    ErrCodeDatabaseError      = "DATABASE_ERROR"
)
```