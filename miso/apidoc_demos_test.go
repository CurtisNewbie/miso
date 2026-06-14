package miso

import (
	"strings"
	"testing"
)

func TestGenJavaHttpClientDemo_PostWithReqAndQuery(t *testing.T) {
	d := HttpRouteDoc{
		Name:   "CreateUser",
		Url:    "/api/user",
		Method: "POST",
		Desc:   "Create a new user",
		JsonRequestDesc: TypeDesc{
			TypeName: "CreateUserReq",
			Fields: []FieldDesc{
				{GoFieldName: "Name", JsonName: "name", OriginTypeName: "string", TypeNameAlias: "string"},
				{GoFieldName: "Age", JsonName: "age", OriginTypeName: "int", TypeNameAlias: "int"},
			},
		},
		JsonResponseDesc: TypeDesc{
			TypeName: "Resp",
			Fields: []FieldDesc{
				{
					GoFieldName: "Data", TypeNameAlias: "CreateUserRes",
					Fields: []FieldDesc{
						{GoFieldName: "UserId", JsonName: "userId", OriginTypeName: "string", TypeNameAlias: "string"},
					},
				},
				{GoFieldName: "Error", OriginTypeName: "bool", TypeNameAlias: "bool"},
				{GoFieldName: "ErrorCode", OriginTypeName: "string", TypeNameAlias: "string"},
			},
		},
		QueryParams: []ParamDoc{
			{Name: "role", Desc: "User role"},
		},
		Headers: []ParamDoc{
			{Name: "Authorization", Desc: "Bearer token"},
		},
	}

	code := GenJavaHttpClientDemo(d, "user-svc")
	if code == "" {
		t.Fatal("expected non-empty generated code")
	}

	// Should contain method signature
	if !strings.Contains(code, "public CreateUserRes createUser(") {
		t.Errorf("expected method signature 'public CreateUserRes createUser(...)', got:\n%s", code)
	}

	// Should contain request body parameter
	if !strings.Contains(code, "CreateUserReq req") {
		t.Errorf("expected 'CreateUserReq req' parameter, got:\n%s", code)
	}

	// Should contain query param
	if !strings.Contains(code, "String role") {
		t.Errorf("expected 'String role' parameter, got:\n%s", code)
	}

	// Should contain header param
	if !strings.Contains(code, "String authorization") {
		t.Errorf("expected 'String authorization' parameter, got:\n%s", code)
	}

	// Should use OkHttpClient
	if !strings.Contains(code, "OkHttpClient") {
		t.Errorf("expected OkHttpClient usage, got:\n%s", code)
	}

	// Should use ObjectMapper
	if !strings.Contains(code, "ObjectMapper") {
		t.Errorf("expected ObjectMapper usage, got:\n%s", code)
	}

	// Should have import okhttp3
	if !strings.Contains(code, "import okhttp3.*;") {
		t.Errorf("expected import okhttp3.*, got:\n%s", code)
	}

	// Should use .post(body)
	if !strings.Contains(code, ".post(body)") {
		t.Errorf("expected .post(body), got:\n%s", code)
	}

	// Should have OkHttpClient client = new OkHttpClient()
	if !strings.Contains(code, "OkHttpClient client = new OkHttpClient();") {
		t.Errorf("expected OkHttpClient client creation, got:\n%s", code)
	}
}

func TestGenJavaHttpClientDemo_GetNoBody(t *testing.T) {
	d := HttpRouteDoc{
		Name:   "GetUser",
		Url:    "/api/user/{id}",
		Method: "GET",
		Desc:   "Get user by id",
		JsonResponseDesc: TypeDesc{
			TypeName: "UserDto",
			Fields: []FieldDesc{
				{GoFieldName: "Name", JsonName: "name", OriginTypeName: "string", TypeNameAlias: "string"},
			},
		},
	}

	code := GenJavaHttpClientDemo(d, "user-svc")
	if code == "" {
		t.Fatal("expected non-empty generated code")
	}

	// No request body param
	if strings.Contains(code, " req,") || strings.Contains(code, " req)") {
		t.Errorf("expected no request body parameter for GET, got:\n%s", code)
	}

	// Should use OkHttp and .get()
	if !strings.Contains(code, ".get()") {
		t.Errorf("expected .get(), got:\n%s", code)
	}

	// Should have import okhttp3
	if !strings.Contains(code, "import okhttp3.*;") {
		t.Errorf("expected import okhttp3.*, got:\n%s", code)
	}

	// Should use ObjectMapper
	if !strings.Contains(code, "ObjectMapper") {
		t.Errorf("expected ObjectMapper usage, got:\n%s", code)
	}

	// Should have try-with-resources
	if !strings.Contains(code, "try (Response") {
		t.Errorf("expected try-with-resources, got:\n%s", code)
	}
}

func TestGenJavaHttpClientDemo_NoResp(t *testing.T) {
	d := HttpRouteDoc{
		Name:   "DeleteUser",
		Url:    "/api/user",
		Method: "DELETE",
		Desc:   "Delete user",
	}

	code := GenJavaHttpClientDemo(d, "user-svc")
	if code == "" {
		t.Fatal("expected non-empty generated code")
	}

	// Should be void return
	if !strings.Contains(code, "public void") {
		t.Errorf("expected void return type, got:\n%s", code)
	}

	// Should use try-with-resources for void (auto-close Response)
	if !strings.Contains(code, "try (Response response") {
		t.Errorf("expected try-with-resources for void method, got:\n%s", code)
	}

	// Should use OkHttp
	if !strings.Contains(code, "OkHttpClient") {
		t.Errorf("expected OkHttpClient usage, got:\n%s", code)
	}

	// Should have import okhttp3
	if !strings.Contains(code, "import okhttp3.*;") {
		t.Errorf("expected import okhttp3.*, got:\n%s", code)
	}

	// Should use .delete()
	if !strings.Contains(code, ".delete()") {
		t.Errorf("expected .delete(), got:\n%s", code)
	}
}

func TestGuessJavaTypeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"string", "String"},
		{"int", "Integer"},
		{"int64", "Long"},
		{"float64", "Double"},
		{"bool", "Boolean"},
		{"CreateUserReq", "CreateUserReq"},
		{"[]string", "List<String>"},
		{"[]CreateUserReq", "List<CreateUserReq>"},
		{"*string", "String"},
		{"miso.PageRes", "PageRes"},
		{"", ""},
	}
	for _, tt := range tests {
		got := guessJavaTypeName(tt.input)
		if got != tt.want {
			t.Errorf("guessJavaTypeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
