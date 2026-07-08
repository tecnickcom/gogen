package passwordhash_test

import (
	"fmt"
	"log"

	"github.com/tecnickcom/gogen/pkg/passwordhash"
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

func ExampleParams_PasswordNeedsRehash() {
	// A hash produced with a weaker cost is detected as needing a rehash by a
	// stronger configuration, so it can be transparently upgraded on next login.
	weak := passwordhash.New(passwordhash.WithTime(1))

	secret := "Example-Password-03"

	hash, err := weak.PasswordHash(secret)
	if err != nil {
		log.Fatal(err)
	}

	strong := passwordhash.New(passwordhash.WithTime(4))

	// The same configuration that produced the hash reports no rehash needed.
	weakNeeds, err := weak.PasswordNeedsRehash(hash)
	if err != nil {
		log.Fatal(err)
	}

	// The stronger configuration reports that the stored hash should be upgraded.
	strongNeeds, err := strong.PasswordNeedsRehash(hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(weakNeeds)
	fmt.Println(strongNeeds)

	// Output:
	// false
	// true
}
