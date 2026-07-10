package passwordhash_test

import (
	"fmt"
	"log"
	"strings"

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

func ExampleWithFormat() {
	// WithFormat(FormatPHC) emits the cross-language PHC string format instead of
	// the default base64 JSON. The same Params verifies it back: the format is
	// auto-detected from the stored value.
	p := passwordhash.New(
		passwordhash.WithFormat(passwordhash.FormatPHC),
		passwordhash.WithMemory(16_384),
		passwordhash.WithThreads(1),
	)

	secret := "Example-Password-04"

	hash, err := p.PasswordHash(secret)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(strings.HasPrefix(hash, "$argon2id$v=19$"))

	ok, err := p.PasswordVerify(secret, hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ok)

	// Output:
	// true
	// true
}

func ExampleParams_PasswordVerify_phc() {
	// A hash produced by another Argon2 implementation in the standard PHC string
	// format is verified transparently, with no configuration change: verification
	// detects the format from the stored value's leading '$'.
	p := passwordhash.New()

	phc := "$argon2id$v=19$m=65536,t=1,p=16$5wnnitUhezr1gnGhyMEU7A$BcbRTU4SCrd14bVS4sqPFbwonv+yiogOnxbV1pQLdV0"

	ok, err := p.PasswordVerify("test", phc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ok)

	// Output:
	// true
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
