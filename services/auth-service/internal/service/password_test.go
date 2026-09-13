package service

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	p := NewPasswordService(Argon2Config{
		Memory:      64 * 1024,
		Iterations:  1,
		Parallelism: 2,
		KeyLength:   32,
		SaltLength:  16,
	})
	hash, err := p.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := p.Verify("correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("expected password to verify: ok=%v err=%v", ok, err)
	}
	ok, err = p.Verify("wrong password", hash)
	if err != nil || ok {
		t.Fatalf("expected wrong password to fail: ok=%v err=%v", ok, err)
	}
}
