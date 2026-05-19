package auth

import "testing"

func TestEnsureSessionReusesPreferredID(t *testing.T) {
	manager := NewAuthManager()

	session, err := manager.EnsureSession("device-1", "preferred-session")
	if err != nil {
		t.Fatalf("EnsureSession trả về lỗi: %v", err)
	}
	if session.ID != "preferred-session" {
		t.Fatalf("mong đợi session id ưu tiên, nhận được %q", session.ID)
	}
	if session.DeviceID != "device-1" {
		t.Fatalf("mong đợi device-1, nhận được %q", session.DeviceID)
	}

	reused, err := manager.EnsureSession("device-2", "preferred-session")
	if err != nil {
		t.Fatalf("EnsureSession trả về lỗi khi tái sử dụng: %v", err)
	}
	if reused != session {
		t.Fatal("mong đợi EnsureSession tái sử dụng object session hiện có")
	}
	if reused.DeviceID != "device-2" {
		t.Fatalf("mong đợi device id của session tái sử dụng được cập nhật, nhận được %q", reused.DeviceID)
	}
}

func TestEnsureSessionCreatesNewSessionWhenPreferredIDEmpty(t *testing.T) {
	manager := NewAuthManager()

	session, err := manager.EnsureSession("device-1", "")
	if err != nil {
		t.Fatalf("EnsureSession trả về lỗi: %v", err)
	}
	if session.ID == "" {
		t.Fatal("mong đợi session id được tạo")
	}
	if session.DeviceID != "device-1" {
		t.Fatalf("mong đợi device-1, nhận được %q", session.DeviceID)
	}
}
