# Validation

Request validation framework for ensuring data integrity and providing meaningful error messages.

**Table of Contents:**
- Overview
- Basic Usage
- Validation Rules
- Combining Rules
- Custom Error Messages
- Parameterized Rules
- Validation Errors
- Advanced Examples
- Validation Flow
- Best Practices
- Validation Rules Reference
- Integration with misoapi
- Testing Validation

## Overview

Miso provides a declarative validation system using struct tags. Validation automatically runs before handler execution, ensuring invalid requests are rejected early with clear error messages.

## Basic Usage

### Validation Tags

Add `valid` tags to struct fields:

```go
type CreateUserReq struct {
    Name     string `json:"name" valid:"notEmpty:Name is required"`
    Email    string `json:"email" valid:"notEmpty:Email is required"`
    Age      int    `json:"age" valid:"positive:Age must be positive"`
    Type     string `json:"type" valid:"member:ADMIN|USER|GUEST"`
    Bio      string `json:"bio" valid:"maxLen:500:Bio cannot exceed 500 characters"`
}
```

### Automatic Validation

When using `AutoHandler`, validation runs automatically:

```go
// misoapi-http: POST /user
func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error) {
    // req is already validated - safe to use
    // Handler implementation
    return CreateUserRes{}, nil
}
```

### Manual Validation

Validate structs manually:

```go
import "github.com/curtisnewbie/miso"

func TestValidate(t *testing.T) {
    v := CreateUserReq{
        Name:  "",  // Invalid: empty
        Email: "",  // Invalid: empty
        Age:   -1,  // Invalid: not positive
    }
    err := miso.Validate(v)
    if err != nil {
        t.Fatal(err)
    }
}
```

## Validation Rules

### notEmpty

Field must not be empty (zero value for type):

```go
type Req struct {
    Name   string `valid:"notEmpty"`
    Items  []int  `valid:"notEmpty"`  // Slice must not be empty
    Count  int    `valid:"notEmpty"`  // Must be != 0
    Config map[string]string `valid:"notEmpty"`  // Map must not be empty
}
```

### notNil

Field pointer must not be nil:

```go
type Req struct {
    User *User `valid:"notNil"`
}
```

### positive

Numeric value must be > 0 (supports int types and string that can be parsed as int):

```go
type Req struct {
    Price int     `valid:"positive"`
    Count float64 `valid:"positive"`
    Age   string  `valid:"positive"`  // String must be a positive integer
}
```

### positiveOrZero

Numeric value must be >= 0 (supports int types and string that can be parsed as int):

```go
type Req struct {
    Discount int    `valid:"positiveOrZero"`
    Count    string `valid:"positiveOrZero"`  // String must be >= 0
}
```

### negative

Numeric value must be < 0 (supports int types and string that can be parsed as int):

```go
type Req struct {
    Temperature int    `valid:"negative"`
    Offset      string `valid:"negative"`  // String must be negative
}
```

### negativeOrZero

Numeric value must be <= 0 (supports int types and string that can be parsed as int):

```go
type Req struct {
    Offset int    `valid:"negativeOrZero"`
    Value  string `valid:"negativeOrZero"`  // String must be <= 0
}
```

### notZero

Field must not be zero value (supports int types and string that can be parsed as int):

```go
type Req struct {
    ID    int    `valid:"notZero"`
    Value string `valid:"notZero"`  // String must not be "0"
}
```

### maxLen

String/array/slice length must not exceed specified value:

```go
type Req struct {
    Title string `valid:"maxLen:100"`
    Tags  []string `valid:"maxLen:10"`  // Array/slice length must not exceed 10
}
```

### member

Value must be one of the specified options:

```go
type Req struct {
    Status string `valid:"member:ACTIVE|INACTIVE|PENDING"`
    Type   string `valid:"member:ENUM_ONE|ENUM_TWO|"`  // Trailing pipe allows empty string
}
```

### trim

Trim whitespace from string before validation:

```go
type Req struct {
    Name string `valid:"trim,notEmpty"`  // Trim first, then check notEmpty
}
```

### validated

Recursively validate nested struct or pointer (also validates each element of slices/arrays):

```go
type CreateUserReq struct {
    Name     string  `valid:"notEmpty"`
    Profile  *Profile `valid:"notNil,validated"`
    Items    []Item   `valid:"validated"`  // Validates each item in the slice
}

type Profile struct {
    Bio  string `valid:"maxLen:500"`
    Age  int    `valid:"positive"`
}
```

## Combining Rules

Multiple rules can be applied using comma separator:

```go
type Req struct {
    Name     string `valid:"maxLen:50,notEmpty:Name is required"`
    Email    string `valid:"notEmpty,member:user@example.com|admin@example.com"`
    Optional *Child `valid:"notNil,validated"`  // Check notNil first, then validate
}
```

Rules are validated in order. If a rule fails, subsequent rules for that field are not evaluated.

## Custom Error Messages

Add custom error messages after the rule:

```go
type Req struct {
    Name  string `valid:"notEmpty:Please provide your name"`
    Email string `valid:"notEmpty:Email address is required"`
    Age   int    `valid:"positive:Age must be a positive number"`
    Bio   string `valid:"maxLen:500:Bio cannot exceed 500 characters"`
}
```

## Parameterized Rules

Some rules accept parameters in the format `[RULE_NAME]:[PARAM]`:

```go
type Req struct {
    Name    string `valid:"maxLen:100"`           // Maximum 100 characters
    Count   int    `valid:"member:1|2|3|4|5"`     // Must be 1, 2, 3, 4, or 5
    Status  string `valid:"member:A|B|C"`         // Must be A, B, or C
}
```

## Validation Errors

When validation fails, the handler receives an error with the first validation failure. The error message format is:

```json
{
  "errorCode": "XXXX",
  "msg": "name must not be empty",
  "error": true,
  "data": null
}
```

The message includes the field name and the validation reason. When using custom error messages, the custom message is used instead:

```go
type Req struct {
    Name string `valid:"notEmpty:Name is required"`
}
// Error message: "Name is required"
```

### Multiple Validation Failures

Currently, only the first validation failure is returned. To see all validation errors, you can manually validate:

```go
func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error) {
    // Manual validation for detailed errors
    if err := miso.Validate(req); err != nil {
        // Handle validation error
        return CreateUserRes{}, err
    }
    // Handler implementation
}
```

## Advanced Examples

### Nested Struct Validation

```go
type CreateUserReq struct {
    Name    string  `valid:"notEmpty"`
    Address Address `valid:"validated"`  // Recursively validate Address
}

type Address struct {
    Street  string `valid:"notEmpty"`
    City    string `valid:"notEmpty"`
    ZipCode string `valid:"maxLen:10"`
}
```

### Slice/Array Element Validation

The `validated` rule also validates each element of slices and arrays:

```go
type CreateOrderReq struct {
    Items []OrderItem `valid:"validated"`  // Validates each OrderItem
}

type OrderItem struct {
    ProductID string `valid:"notEmpty"`
    Quantity  int    `valid:"positive"`
}
```

### Optional Nested Fields

```go
type UpdateUserReq struct {
    ID      string    `valid:"notEmpty"`
    Profile *Profile  `valid:"notNil,validated"`  // Must not be nil, then validate
}

type Profile struct {
    Bio  string `valid:"maxLen:500"`
    Age  int    `valid:"positive"`
}
```

### Complex Validation

```go
type OrderReq struct {
    ProductID    string  `valid:"notEmpty"`
    Quantity     int     `valid:"positive"`
    Price        float64 `valid:"positiveOrZero"`
    DiscountCode string  `valid:"trim,member:PROMO10|PROMO20|"`  // Trim and check membership
    Shipping     Address `valid:"validated"`
    Billing      Address `valid:"validated"`
}
```

## Validation Flow

1. **Request Parsing**: Request body/query/headers are parsed into struct
2. **Validation**: `valid` tags are evaluated in order
3. **First Failure**: If any validation fails, error is returned immediately
4. **Handler Execution**: If all validations pass, handler is called with validated data

## Best Practices

### 1. Use Descriptive Error Messages

```go
// Good
type Req struct {
    Email string `valid:"notEmpty:Please provide a valid email address"`
}

// Bad
type Req struct {
    Email string `valid:"notEmpty"`  // Generic message
}
```

### 2. Order Rules by Specificity

```go
// Good - Check length first, then notEmpty
type Req struct {
    Name string `valid:"maxLen:100,notEmpty:Name is required"`
}

// Works the same but conceptually clearer
type Req struct {
    Name string `valid:"notEmpty,maxLen:100"`
}
```

### 3. Use Trim for String Inputs

```go
type Req struct {
    Name string `valid:"trim,notEmpty"`  // Trim whitespace before checking
    Bio  string `valid:"trim,maxLen:500"`  // Trim before checking length
}
```

### 4. Validate at API Boundaries

Always validate at the API handler level, not in service layer:

```go
// Good - Validate in handler
func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error) {
    // req is validated by framework
    return userService.Create(req), nil
}
```

## Validation Rules Reference

| Rule | Description | Parameters | Supported Types |
|------|-------------|------------|-----------------|
| `notEmpty` | Field must not be empty | None | string, array, slice, map |
| `notNil` | Pointer must not be nil | None | pointer, slice, map, func |
| `notZero` | Field must not be zero value | None | int types, string |
| `positive` | Value must be > 0 | None | int types, string |
| `positiveOrZero` | Value must be >= 0 | None | int types, string |
| `negative` | Value must be < 0 | None | int types, string |
| `negativeOrZero` | Value must be <= 0 | None | int types, string |
| `maxLen` | Length must not exceed value | `maxLen:N` | string, array, slice |
| `member` | Value must be in specified list | `member:A\|B\|C` | string |
| `trim` | Trim whitespace from string | None | string, *string |
| `validated` | Recursively validate nested struct | None | struct, pointer, slice, array |

## Integration with misoapi

Validation works seamlessly with `misoapi` code generation:

```go
// misoapi-http: POST /user
// misoapi-desc: Create new user with validation
func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error) {
    // req is automatically validated
    return CreateUserRes{}, nil
}

type CreateUserReq struct {
    Name  string `json:"name" valid:"notEmpty:Name is required"`
    Email string `json:"email" valid:"notEmpty:Email is required"`
    Age   int    `json:"age" valid:"positive:Age must be positive"`
}
```

## Testing Validation

Test validation logic in isolation:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        req     CreateUserReq
        wantErr bool
    }{
        {"valid", CreateUserReq{Name: "John", Email: "john@example.com"}, false},
        {"empty name", CreateUserReq{Name: "", Email: "john@example.com"}, true},
        {"empty email", CreateUserReq{Name: "John", Email: ""}, true},
        {"negative age", CreateUserReq{Name: "John", Email: "john@example.com", Age: -1}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := miso.Validate(tt.req)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```