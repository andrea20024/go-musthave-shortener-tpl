package storage

import (
	"errors"
	"testing"
)

func TestNewMapRepository(t *testing.T) {
	repo := NewMapRepository()
	if repo == nil {
		t.Fatal("NewMapRepository returned nil")
	}
}

func TestAdd_Get(t *testing.T) {
	repo := NewMapRepository()

	err := repo.Add("abc", "https://example.com", "user1")
	if err != nil {
		t.Fatalf("Add() error = %v, want nil", err)
	}

	url, err := repo.Get("abc")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if url != "https://example.com" {
		t.Errorf("Get() = %q, want %q", url, "https://example.com")
	}
}

func TestAdd_DuplicateURL(t *testing.T) {
	repo := NewMapRepository()

	if err := repo.Add("key1", "https://example.com", "user1"); err != nil {
		t.Fatalf("first Add() error = %v, want nil", err)
	}

	err := repo.Add("key2", "https://example.com", "user2")
	if err == nil {
		t.Fatal("second Add() error = nil, want DuplicateError")
	}
	var dupErr *DuplicateError
	if !errors.As(err, &dupErr) {
		t.Errorf("error = %T, want *DuplicateError", err)
	}
	if err.Error() != "duplicate" {
		t.Errorf("error message = %q, want %q", err.Error(), "duplicate")
	}
}

func TestGet_NotFound(t *testing.T) {
	repo := NewMapRepository()

	_, err := repo.Get("nonexistent")
	if err == nil {
		t.Fatal("Get() error = nil, want non-nil")
	}
}

func TestGetKeyByURL(t *testing.T) {
	repo := NewMapRepository()

	repo.Add("short1", "https://example.com", "user1")

	key, err := repo.GetKeyByURL("https://example.com")
	if err != nil {
		t.Fatalf("GetKeyByURL() error = %v, want nil", err)
	}
	if key != "short1" {
		t.Errorf("GetKeyByURL() = %q, want %q", key, "short1")
	}
}

func TestGetKeyByURL_NotFound(t *testing.T) {
	repo := NewMapRepository()

	_, err := repo.GetKeyByURL("https://nonexistent.com")
	if err == nil {
		t.Fatal("GetKeyByURL() error = nil, want non-nil")
	}
}

func TestGetKeyByURL_Deleted(t *testing.T) {
	repo := NewMapRepository()

	repo.Add("short1", "https://example.com", "user1")
	repo.DeleteUserURLs("user1", []string{"short1"})

	_, err := repo.GetKeyByURL("https://example.com")
	if err == nil {
		t.Fatal("GetKeyByURL() for deleted URL error = nil, want non-nil")
	}
}

func TestGet_Deleted(t *testing.T) {
	repo := NewMapRepository()

	repo.Add("short1", "https://example.com", "user1")
	repo.DeleteUserURLs("user1", []string{"short1"})

	_, err := repo.Get("short1")
	if err == nil {
		t.Fatal("Get() for deleted URL error = nil, want DeletedError")
	}
	if _, ok := err.(*DeletedError); !ok {
		t.Errorf("error = %T, want *DeletedError", err)
	}
}

func TestGetUserURLs(t *testing.T) {
	repo := NewMapRepository()

	repo.Add("url1", "https://example.com/1", "user1")
	repo.Add("url2", "https://example.com/2", "user1")
	repo.Add("url3", "https://example.com/3", "user2")

	urls, err := repo.GetUserURLs("user1")
	if err != nil {
		t.Fatalf("GetUserURLs() error = %v, want nil", err)
	}
	if len(urls) != 2 {
		t.Errorf("GetUserURLs() len = %d, want 2", len(urls))
	}

	urls2, err := repo.GetUserURLs("user2")
	if err != nil {
		t.Fatalf("GetUserURLs(user2) error = %v, want nil", err)
	}
	if len(urls2) != 1 {
		t.Errorf("GetUserURLs(user2) len = %d, want 1", len(urls2))
	}
}

func TestGetUserURLs_DeletedNotIncluded(t *testing.T) {
	repo := NewMapRepository()

	repo.Add("url1", "https://example.com/1", "user1")
	repo.Add("url2", "https://example.com/2", "user1")
	repo.DeleteUserURLs("user1", []string{"url1"})

	urls, err := repo.GetUserURLs("user1")
	if err != nil {
		t.Fatalf("GetUserURLs() error = %v, want nil", err)
	}
	if len(urls) != 1 {
		t.Errorf("GetUserURLs() len = %d, want 1 (deleted excluded)", len(urls))
	}
	if urls[0].ShortURL != "url2" {
		t.Errorf("GetUserURLs()[0].ShortURL = %q, want %q", urls[0].ShortURL, "url2")
	}
}

func TestGetUserURLs_EmptyUser(t *testing.T) {
	repo := NewMapRepository()

	urls, err := repo.GetUserURLs("nobody")
	if err != nil {
		t.Fatalf("GetUserURLs(nobody) error = %v, want nil", err)
	}
	if len(urls) != 0 {
		t.Errorf("GetUserURLs(nobody) len = %d, want 0", len(urls))
	}
}

func TestDeleteUserURLs_NonExistentUser(t *testing.T) {
	repo := NewMapRepository()

	err := repo.DeleteUserURLs("nonexistent", []string{"key1"})
	if err != nil {
		t.Errorf("DeleteUserURLs(nonexistent) error = %v, want nil", err)
	}
}

func TestDeleteUserURLs_NonExistentKey(t *testing.T) {
	repo := NewMapRepository()

	repo.Add("url1", "https://example.com/1", "user1")

	err := repo.DeleteUserURLs("user1", []string{"nonexistent"})
	if err != nil {
		t.Errorf("DeleteUserURLs(nonexistent key) error = %v, want nil", err)
	}

	url, err := repo.Get("url1")
	if err != nil {
		t.Errorf("Get() after partial delete error = %v", err)
	}
	if url != "https://example.com/1" {
		t.Errorf("Get() = %q, want %q", url, "https://example.com/1")
	}
}

func TestAddBatch(t *testing.T) {
	repo := NewMapRepository()

	urls := map[string]string{
		"batch1": "https://example.com/1",
		"batch2": "https://example.com/2",
		"batch3": "https://example.com/3",
	}

	err := repo.AddBatch(urls, "user1")
	if err != nil {
		t.Fatalf("AddBatch() error = %v, want nil", err)
	}

	for key, expected := range urls {
		val, err := repo.Get(key)
		if err != nil {
			t.Errorf("Get(%q) error = %v, want nil", key, err)
		}
		if val != expected {
			t.Errorf("Get(%q) = %q, want %q", key, val, expected)
		}
	}
}

func TestAddBatch_Duplicate(t *testing.T) {
	repo := NewMapRepository()

	repo.Add("standalone", "https://example.com/exist", "user1")

	urls := map[string]string{
		"batch1": "https://example.com/exist",
		"batch2": "https://example.com/new",
	}

	err := repo.AddBatch(urls, "user1")
	if err == nil {
		t.Fatal("AddBatch() with duplicate error = nil, want error")
	}
	var dupErr *DuplicateError
	if !errors.As(err, &dupErr) {
		t.Errorf("error = %T, want *DuplicateError", err)
	}
}

func TestPing(t *testing.T) {
	repo := NewMapRepository()

	err := repo.Ping()
	if err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestIsDuplicateError(t *testing.T) {
	repo := NewMapRepository()

	err := &DuplicateError{key: "test", url: "https://example.com"}
	if !repo.IsDuplicateError(err) {
		t.Error("IsDuplicateError() = false, want true")
	}
}

func TestIsDuplicateError_NonDuplicate(t *testing.T) {
	repo := NewMapRepository()

	err := errors.New("some other error")
	if repo.IsDuplicateError(err) {
		t.Error("IsDuplicateError() = true, want false for non-DuplicateError")
	}
}

func TestIsDuplicateError_Nil(t *testing.T) {
	repo := NewMapRepository()

	if repo.IsDuplicateError(nil) {
		t.Error("IsDuplicateError(nil) = true, want false")
	}
}

func TestIsDeletedError(t *testing.T) {
	repo := NewMapRepository()

	err := &DeletedError{}
	if !repo.IsDeletedError(err) {
		t.Error("IsDeletedError() = false, want true")
	}
}

func TestIsDeletedError_NonDeleted(t *testing.T) {
	repo := NewMapRepository()

	err := errors.New("some other error")
	if repo.IsDeletedError(err) {
		t.Error("IsDeletedError() = true, want false for non-DeletedError")
	}
}

func TestIsDeletedError_Nil(t *testing.T) {
	repo := NewMapRepository()

	if repo.IsDeletedError(nil) {
		t.Error("IsDeletedError(nil) = true, want false")
	}
}

func TestDuplicateError_Error(t *testing.T) {
	err := &DuplicateError{key: "test", url: "https://example.com"}
	if err.Error() != "duplicate" {
		t.Errorf("DuplicateError.Error() = %q, want %q", err.Error(), "duplicate")
	}
}

func TestDeletedError_Error(t *testing.T) {
	err := &DeletedError{}
	if err.Error() != "deleted" {
		t.Errorf("DeletedError.Error() = %q, want %q", err.Error(), "deleted")
	}
}

func TestErrKeyNotFound(t *testing.T) {
	if ErrKeyNotFound == nil {
		t.Fatal("ErrKeyNotFound is nil, want non-nil")
	}
	if ErrKeyNotFound.Error() != "key not found" {
		t.Errorf("ErrKeyNotFound.Error() = %q, want %q", ErrKeyNotFound.Error(), "key not found")
	}
}

func TestConcurrent_Add(t *testing.T) {
	repo := NewMapRepository()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "key" + string(rune(id))
			url := "https://example.com/" + string(rune(id))
			if err := repo.Add(key, url, "user1"); err != nil {
				t.Errorf("Add(%d) error = %v", id, err)
			}
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestUserURL_Struct(t *testing.T) {
	u := UserURL{
		ShortURL:    "abc",
		OriginalURL: "https://example.com",
	}
	if u.ShortURL != "abc" {
		t.Errorf("ShortURL = %q, want %q", u.ShortURL, "abc")
	}
	if u.OriginalURL != "https://example.com" {
		t.Errorf("OriginalURL = %q, want %q", u.OriginalURL, "https://example.com")
	}
}
