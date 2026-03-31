// Logger 包内测：与实现同包以测试未导出函数；断言优先读临时文件中的 JSON 行。
// 说明：init 里 ConsoleWriter 绑定的是进程启动时的 os.Stdout，go test 时终端可能仍有 INFO 输出。
package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFormatFieldValue_multilineAndJSON 验证控制台字段格式化：多行前缀换行、含空格字符串加引号、JSON 形态保留等。
func TestFormatFieldValue_multilineAndJSON(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   any
		want string
	}{
		{name: "plain", in: "x", want: "x"},
		{name: "multiline", in: "a\nb", want: "\na\nb"},
		{name: "json_object", in: `{"a":1}`, want: `{"a":1}`},
		{name: "json_array_bracket_form", in: `[1 2]`, want: `[1 2]`},
		{name: "bytes", in: []byte("hi"), want: "hi"},
		{name: "other", in: 42, want: "42"},
		{name: "quoted_string_unquote", in: `"hello"`, want: "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatFieldValue(tc.in)
			if got != tc.want {
				t.Fatalf("formatFieldValue(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestMaskSecrets_redactsBotToken 验证 Telegram bot token 中间段被掩码，首尾前缀保留便于排查。
func TestMaskSecrets_redactsBotToken(t *testing.T) {
	t.Parallel()
	raw := "prefix bot123456:abcdEFGHijklmnopQRstuVWXyz9876suffix"
	got := maskSecrets(raw)
	if strings.Contains(got, "ijklmnopQRstuVWX") {
		t.Fatalf("expected secret middle redacted, got %q", got)
	}
	if !strings.Contains(got, "bot123456:") || !strings.Contains(got, "abcd") {
		t.Fatalf("expected prefix preserved, got %q", got)
	}
	if !strings.Contains(got, "****") {
		t.Fatalf("expected placeholder, got %q", got)
	}
}

// TestMaskSecrets_noToken 无 token 时原文不变。
func TestMaskSecrets_noToken(t *testing.T) {
	t.Parallel()
	s := "no secret here"
	if maskSecrets(s) != s {
		t.Fatalf("got %q", maskSecrets(s))
	}
}

// TestGetLevel_SetLevel 验证业务阈值 currentLevel 与 zerolog 全局等级同步更新。
func TestGetLevel_SetLevel(t *testing.T) {
	t.Cleanup(func() {
		SetLevel(INFO)
	})
	SetLevel(DEBUG)
	if GetLevel() != DEBUG {
		t.Fatalf("GetLevel = %v", GetLevel())
	}
	SetLevel(WARN)
	if GetLevel() != WARN {
		t.Fatalf("GetLevel = %v", GetLevel())
	}
}

// TestDebug_suppressedWhenLevelInfo 当 currentLevel 为 INFO 时，DEBUG 不应写入文件（也不应产生文件行）。
func TestDebug_suppressedWhenLevelInfo(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		DisableFileLogging()
		SetLevel(INFO)
	})

	SetLevel(INFO)
	Debug("should-not-appear")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "should-not-appear") {
		t.Fatalf("debug leaked: %s", data)
	}
}

// TestInfo_writesJSONLineToFile 启用文件日志后，Info 会追加一行可解析的 JSON，并包含 message 与 caller 等字段。
func TestInfo_writesJSONLineToFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "out.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		DisableFileLogging()
		SetLevel(INFO)
	})
	SetLevel(INFO)

	msg := "hello-json-" + t.Name()
	Info(msg)

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSpace(string(bytes.TrimSpace(b)))
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, line)
	}
	if obj["message"] != msg {
		t.Fatalf("message field: got %v", obj["message"])
	}
}

// TestInfoF_fieldTypes appendFields 对 string/int/bool 等写入 JSON 后类型与取值正确。
func TestInfoF_fieldTypes(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "fields.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		DisableFileLogging()
		SetLevel(INFO)
	})
	SetLevel(INFO)

	InfoF("typed", map[string]any{
		"s": "a",
		"i": 7,
		"b": true,
	})

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(raw), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["s"] != "a" {
		t.Fatalf("s: %v", obj["s"])
	}
	// JSON numbers are float64
	if fi, ok := obj["i"].(float64); !ok || fi != 7 {
		t.Fatalf("i: %v", obj["i"])
	}
	if obj["b"] != true {
		t.Fatalf("b: %v", obj["b"])
	}
}

// TestEnableFileLogging_createsNestedDir 验证日志路径父目录不存在时会 MkdirAll 并成功写入。
func TestEnableFileLogging_createsNestedDir(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nested", "a.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		DisableFileLogging()
	})
	SetLevel(INFO)
	Info("nested-dir")
	b, err := os.ReadFile(logPath)
	if err != nil || !strings.Contains(string(b), "nested-dir") {
		t.Fatalf("read err=%v body=%q", err, b)
	}
}

// TestDisableFileLogging_stopsFileWrites 关闭文件后 fileLogger 不应再接收内容（比对前后文件大小或内容）。
func TestDisableFileLogging_stopsFileWrites(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "onoff.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	SetLevel(INFO)
	Info("one")
	DisableFileLogging()
	sizeAfterDisable, _ := os.Stat(logPath)
	first := sizeAfterDisable.Size()
	Info("two-should-not-be-in-file")
	second, _ := os.Stat(logPath)
	if second.Size() != first {
		t.Fatalf("file grew after disable: %d -> %d", first, second.Size())
	}
}

// TestInfoC_component 验证 component 字段写入 JSON。
func TestInfoC_component(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "c.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		DisableFileLogging()
		SetLevel(INFO)
	})
	SetLevel(INFO)
	InfoC("svc", "with-component")
	var obj map[string]any
	raw, _ := os.ReadFile(logPath)
	if err := json.Unmarshal(bytes.TrimSpace(raw), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["component"] != "svc" {
		t.Fatalf("component: %v", obj["component"])
	}
}

// TestNewLogger_thirdParty_methods 第三方 Logger 封装应带 component 且 maskSecrets 仍可工作（不写 Fatal）。
func TestNewLogger_thirdParty_methods(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "3rd.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		DisableFileLogging()
		SetLevel(INFO)
	})
	SetLevel(INFO)

	l := NewLogger("cmp")
	l.Infof("n=%d", 3)
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(raw), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["component"] != "cmp" {
		t.Fatalf("component %v", obj["component"])
	}
}

// TestLogger_Log_levelMapping 验证 Log 使用 WithLevels 时可将第三方整型等级映射到内部 LogLevel。
func TestLogger_Log_levelMapping(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "map.log")
	if err := EnableFileLogging(logPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		DisableFileLogging()
		SetLevel(INFO)
	})
	SetLevel(DEBUG)

	const fakeWarn = 99
	l := NewLogger("x").WithLevels(map[int]LogLevel{fakeWarn: WARN})
	l.Log(fakeWarn, 0, "mapped-warn")

	raw, _ := os.ReadFile(logPath)
	sc := bufio.NewScanner(bytes.NewReader(raw))
	var last string
	for sc.Scan() {
		last = sc.Text()
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(last), &obj); err != nil {
		t.Fatal(err)
	}
	if level, ok := obj["level"].(string); !ok || !strings.EqualFold(level, "warn") {
		t.Fatalf("level field: %v", obj["level"])
	}
}

// TestLogger_Sync 兼容接口 Sync 应返回 nil。
func TestLogger_Sync(t *testing.T) {
	l := NewLogger("s")
	if err := l.Sync(); err != nil {
		t.Fatal(err)
	}
}

// TestGetCallerSkip_returnsPositive 在测试调用栈下 getCallerSkip 应返回合理 skip（不依赖具体帧号，仅 sanity）。
func TestGetCallerSkip_returnsPositive(t *testing.T) {
	skip := getCallerSkip()
	if skip < 1 || skip > 20 {
		t.Fatalf("unexpected skip %d", skip)
	}
}

// TestFatal_exitsWithCodeOne Fatal 会 os.Exit(1)，在子进程中验证以避免终止当前测试进程。
func TestFatal_exitsWithCodeOne(t *testing.T) {
	if os.Getenv("LOGGER_TEST_FATAL_CHILD") != "" {
		SetLevel(INFO)
		dir := t.TempDir()
		_ = EnableFileLogging(filepath.Join(dir, "fatal.log"))
		Fatal("child-exit")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_exitsWithCodeOne", "-test.count=1")
	cmd.Env = append(os.Environ(), "LOGGER_TEST_FATAL_CHILD=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-nil exit error")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %v stderr=%s", err, stderr.String())
	}
	if code := ee.ExitCode(); code != 1 {
		t.Fatalf("exit code %d, stderr=%s", code, stderr.String())
	}
}
