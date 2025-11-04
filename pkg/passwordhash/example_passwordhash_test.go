package passwordhash_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/passwordhash"
)

func ExampleParams_PasswordVerify() {
	opts := []passwordhash.Option{
		passwordhash.WithKeyLen(32),
		passwordhash.WithSaltLen(16),
		passwordhash.WithTime(3),
		passwordhash.WithMemory(16_384),
		passwordhash.WithThreads(1),
		passwordhash.WithMinPasswordLength(16),
		passwordhash.WithMaxPasswordLength(128),
	}

	p := passwordhash.New(opts...)

	secret := "Example-Password-01"

	hash, err := p.PasswordHash(secret)
	if err != nil {
		log.Fatal(err)
	}

	ok, err := p.PasswordVerify(secret, hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ok)

	ok, err = p.PasswordVerify("Example-Wrong-Password-01", hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ok)

	// Output:
	// true
	// false
}

func ExampleParams_EncryptPasswordVerify() {
	opts := []passwordhash.Option{
		passwordhash.WithKeyLen(32),
		passwordhash.WithSaltLen(16),
		passwordhash.WithTime(3),
		passwordhash.WithMemory(16_384),
		passwordhash.WithThreads(1),
		passwordhash.WithMinPasswordLength(16),
		passwordhash.WithMaxPasswordLength(128),
	}

	p := passwordhash.New(opts...)

	key := []byte("0123456789012345")

	secret := "Example-Password-02"

	hash, err := p.EncryptPasswordHash(key, secret)
	if err != nil {
		log.Fatal(err)
	}

	ok, err := p.EncryptPasswordVerify(key, secret, hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ok)

	ok, err = p.EncryptPasswordVerify(key, "Example-Wrong-Password-02", hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ok)

	// Output:
	// true
	// false
}
