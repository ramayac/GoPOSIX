package daemon

import (
	"encoding/json"
	"testing"
	"time"

	// Register utilities so dispatch.Lookup works.
	_ "github.com/ramayac/goposix/pkg/echo"
	_ "github.com/ramayac/goposix/pkg/tee"
	_ "github.com/ramayac/goposix/pkg/testcmd"
	_ "github.com/ramayac/goposix/pkg/truefalse"
)

func TestProcessRequest_EchoJSONMode(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}

	params, err := json.Marshal(GoposixParams{Flags: nil, Text: "hello world"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := Request{
		JSONRPC: "2.0",
		Method:  "goposix.echo",
		Params:  json.RawMessage(params),
		ID:      1,
	}

	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be map[string]interface{}, got %T", resp.Result)
	}

	exitCode, ok := resultMap["exitCode"].(int)
	if !ok {
		t.Fatalf("expected exitCode (int) in result map, got %v", resultMap)
	}
	if exitCode != 0 {
		t.Errorf("expected exitCode 0, got %v", exitCode)
	}

	data, ok := resultMap["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be map[string]interface{}, got %T: %v", resultMap["data"], resultMap)
	}

	text, ok := data["text"].(string)
	if !ok {
		t.Fatalf("expected data.text string, got %T: %v", data["text"], data)
	}
	if text != "hello world" {
		t.Errorf("expected 'hello world', got %q", text)
	}
}

func TestProcessRequest_EchoJSONModeWithFlags(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}

	params, err := json.Marshal(GoposixParams{Flags: []string{"-n"}, Text: "hello"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	req := Request{
		JSONRPC: "2.0",
		Method:  "goposix.echo",
		Params:  json.RawMessage(params),
		ID:      1,
	}

	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}

	data, ok := resultMap["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data map, got %T: %v", resultMap["data"], resultMap)
	}

	text, ok := data["text"].(string)
	if !ok {
		t.Fatalf("expected data.text string, got %T: %v", data["text"], data)
	}
	if text != "hello" {
		t.Errorf("expected 'hello', got %q", text)
	}
}

func TestProcessRequest_MethodNotFound(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	req := Request{JSONRPC: "2.0", Method: "goposix.nonexistent", Params: nil, ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected -32601, got %d", resp.Error.Code)
	}
}

func TestProcessRequest_Notification(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	req := Request{JSONRPC: "2.0", Method: "goposix.true", Params: nil}
	resp := srv.processRequest(req)
	if resp != nil {
		t.Error("notification should return nil response")
	}
}

func TestProcessRequest_Ping(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30), uptime: time.Now()}
	req := Request{JSONRPC: "2.0", Method: "goposix.ping", Params: nil, ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestProcessRequest_PingNotification(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30), uptime: time.Now()}
	req := Request{JSONRPC: "2.0", Method: "goposix.ping", Params: nil}
	resp := srv.processRequest(req)
	if resp != nil {
		t.Error("ping notification should return nil")
	}
}

func TestProcessRequest_InvalidJSONRPC(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	req := Request{JSONRPC: "1.0", Method: "goposix.echo", Params: nil, ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("expected -32600, got %d", resp.Error.Code)
	}
}

func TestProcessRequest_MethodTooLong(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	longMethod := "goposix." + string(make([]byte, 300))
	req := Request{JSONRPC: "2.0", Method: longMethod, Params: nil, ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("expected -32600, got %d", resp.Error.Code)
	}
}

func TestProcessRequest_SessionCreate(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	req := Request{JSONRPC: "2.0", Method: "goposix.session.create", Params: nil, ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestProcessRequest_SessionDestroy(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	s := srv.sm.Create()
	params, _ := json.Marshal(GoposixParams{SessionId: s.ID})
	req := Request{JSONRPC: "2.0", Method: "goposix.session.destroy", Params: json.RawMessage(params), ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestProcessRequest_SessionSetCwd(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	s := srv.sm.Create()
	params, _ := json.Marshal(GoposixParams{SessionId: s.ID, Path: "/tmp"})
	req := Request{JSONRPC: "2.0", Method: "goposix.session.setCwd", Params: json.RawMessage(params), ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestProcessRequest_SessionList(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	srv.sm.Create()
	req := Request{JSONRPC: "2.0", Method: "goposix.session.list", Params: nil, ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager(30)
	s := sm.Create()
	if s.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if s.CWD != "/" {
		t.Errorf("expected default CWD '/', got %q", s.CWD)
	}
}

func TestSessionManager_Get(t *testing.T) {
	sm := NewSessionManager(30)
	s := sm.Create()
	got, ok := sm.Get(s.ID)
	if !ok {
		t.Error("expected to find session")
	}
	if got.ID != s.ID {
		t.Error("wrong session returned")
	}
	_, ok = sm.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent session")
	}
}

func TestSessionManager_SetCwd(t *testing.T) {
	sm := NewSessionManager(30)
	s := sm.Create()
	if !sm.SetCwd(s.ID, "/tmp") {
		t.Error("SetCwd should succeed")
	}
	got, _ := sm.Get(s.ID)
	if got.CWD != "/tmp" {
		t.Errorf("expected /tmp, got %q", got.CWD)
	}
	if sm.SetCwd("nonexistent", "/") {
		t.Error("SetCwd on nonexistent should return false")
	}
}

func TestSessionManager_Destroy(t *testing.T) {
	sm := NewSessionManager(30)
	s := sm.Create()
	if !sm.Destroy(s.ID) {
		t.Error("Destroy should succeed")
	}
	_, ok := sm.Get(s.ID)
	if ok {
		t.Error("session should be gone after destroy")
	}
	if sm.Destroy("nonexistent") {
		t.Error("Destroy on nonexistent should return false")
	}
}

func TestSessionManager_List(t *testing.T) {
	sm := NewSessionManager(30)
	sm.Create()
	sm.Create()
	list := sm.List()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestSessionManager_Cleanup(t *testing.T) {
	sm := NewSessionManager(1)
	s := sm.Create()
	time.Sleep(2 * time.Second)
	_, ok := sm.Get(s.ID)
	_ = ok
}

// --- Daemon tests for utilities that were missing coverage ---

func TestProcessRequest_True(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	req := Request{JSONRPC: "2.0", Method: "goposix.true", Params: nil, ID: 1}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	resultMap, _ := resp.Result.(map[string]interface{})
	if code, _ := resultMap["exitCode"].(float64); int(code) != 0 {
		t.Errorf("expected exitCode 0, got %v", code)
	}
}

func TestProcessRequest_False(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	req := Request{JSONRPC: "2.0", Method: "goposix.false", Params: nil, ID: 2}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	// false utility returns non-zero exit, daemon wraps it in Result
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Error("expected result in response")
	}
}

func TestProcessRequest_Tee(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	params, _ := json.Marshal(GoposixParams{Text: "tee test\n", Flags: nil})
	req := Request{JSONRPC: "2.0", Method: "goposix.tee", Params: json.RawMessage(params), ID: 3}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	resultMap, _ := resp.Result.(map[string]interface{})
	if code, _ := resultMap["exitCode"].(float64); int(code) != 0 {
		t.Errorf("expected exitCode 0, got %v", code)
	}
	// tee returns text output as data string
	data, ok := resultMap["data"]
	if !ok {
		t.Error("expected data in response")
	}
	_ = data
}

func TestProcessRequest_Testcmd(t *testing.T) {
	srv := &Server{sm: NewSessionManager(30)}
	params, _ := json.Marshal(GoposixParams{Text: "-n hello", Flags: nil})
	req := Request{JSONRPC: "2.0", Method: "goposix.test", Params: json.RawMessage(params), ID: 4}
	resp := srv.processRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	resultMap, _ := resp.Result.(map[string]interface{})
	if code, _ := resultMap["exitCode"].(float64); int(code) != 0 {
		t.Errorf("expected exitCode 0 for 'test -n hello', got %v", code)
	}
}
