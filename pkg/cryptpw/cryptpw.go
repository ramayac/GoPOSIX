// Package cryptpw implements the POSIX-compliant cryptpw utility.
package cryptpw

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"github.com/sergeymakinen/go-crypt/des"
	crypthash "github.com/sergeymakinen/go-crypt/hash"
	"github.com/tredoe/crypt/md5_crypt"
	"github.com/tredoe/crypt/sha256_crypt"
	"github.com/tredoe/crypt/sha512_crypt"
)

var spec = common.FlagSpec{
	Defs: []common.FlagDef{
		{Short: "m", Long: "method", Type: common.FlagValue},
		{Short: "S", Long: "salt", Type: common.FlagValue},
		{Short: "h", Long: "help", Type: common.FlagBool},
		{Long: "json", Type: common.FlagBool},
	},
}

func init() {
	dispatch.Register(dispatch.Command{
		Name:  "cryptpw",
		Usage: "Hash a password using Unix crypt hashes",
		Run:   run,
	})
}

// CryptpwResult represents the JSON response structure.
type CryptpwResult struct {
	Password string `json:"password"`
	Method   string `json:"method"`
	Salt     string `json:"salt"`
	Hash     string `json:"hash"`
}

type desScheme struct {
	Salt []byte `hash:"length:2,inline"`
	Sum  [11]byte
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int {
	jsonMode := false
	for _, arg := range args {
		if arg == "--json" {
			jsonMode = true
			break
		}
	}

	flags, err := common.ParseFlags(args, spec)
	if err != nil {
		if jsonMode {
			common.RenderError("cryptpw", 1, "FLAG_ERROR", err.Error(), true, stderr)
		} else {
			fmt.Fprintf(stderr, "cryptpw: %v\n", err)
		}
		return 1
	}

	if flags.Has("h") || flags.Has("help") {
		helpText := "Usage: cryptpw [-m TYPE] [-S SALT] [PASSWORD] [SALT]\n\n" +
			"Hash a password using Unix crypt hashes.\n\n" +
			"Options:\n" +
			"  -m, --method   Hash method: des, md5, sha256, sha512 (default: sha256)\n" +
			"  -S, --salt     Salt string\n" +
			"  -h, --help     Print help"
		common.Render("cryptpw", struct {
			Help string `json:"help"`
		}{Help: helpText}, jsonMode, stdout, func() {
			fmt.Fprintln(stdout, helpText)
		})
		return 0
	}

	method := flags.Get("m")
	if method == "" {
		method = "sha256"
	}
	method = strings.ToLower(method)

	// Resolve salt from -S or second positional arg
	var rawSalt string
	if flags.Has("S") {
		rawSalt = flags.Get("S")
	}

	var password string
	pos := flags.Positional
	if len(pos) > 0 {
		password = pos[0]
		if len(pos) > 1 {
			rawSalt = pos[1]
		}
	} else {
		// Read password from stdin
		scanner := bufio.NewScanner(stdin)
		if scanner.Scan() {
			password = scanner.Text()
		}
	}

	// Default salt if empty
	if rawSalt == "" {
		if method == "des" {
			rawSalt = "xx"
		} else {
			rawSalt = "12345678"
		}
	}

	var hash string
	switch method {
	case "des":
		saltPart := rawSalt
		if len(saltPart) > 2 {
			saltPart = saltPart[:2]
		}
		key, err := des.Key([]byte(password), []byte(saltPart))
		if err != nil {
			if jsonMode {
				common.RenderError("cryptpw", 1, "HASH_FAILED", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "cryptpw: des hashing failed: %v\n", err)
			}
			return 1
		}
		var sum [11]byte
		crypthash.BigEndianEncoding.Encode(sum[:], key)
		scheme := desScheme{
			Salt: []byte(saltPart),
			Sum:  sum,
		}
		s, err := crypthash.Marshal(scheme)
		if err != nil {
			if jsonMode {
				common.RenderError("cryptpw", 1, "HASH_FAILED", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "cryptpw: des marshaling failed: %v\n", err)
			}
			return 1
		}
		hash = s

	case "md5":
		var salt string
		if strings.HasPrefix(rawSalt, "$1$") {
			salt = rawSalt
		} else {
			saltPart := rawSalt
			if len(saltPart) > 8 {
				saltPart = saltPart[:8]
			}
			salt = "$1$" + saltPart
		}
		res, err := md5_crypt.New().Generate([]byte(password), []byte(salt))
		if err != nil {
			if jsonMode {
				common.RenderError("cryptpw", 1, "HASH_FAILED", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "cryptpw: md5 hashing failed: %v\n", err)
			}
			return 1
		}
		hash = res

	case "sha256":
		salt := buildModularSalt("$5$", rawSalt, 16)
		res, err := sha256_crypt.New().Generate([]byte(password), []byte(salt))
		if err != nil {
			if jsonMode {
				common.RenderError("cryptpw", 1, "HASH_FAILED", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "cryptpw: sha256 hashing failed: %v\n", err)
			}
			return 1
		}
		hash = res

	case "sha512":
		salt := buildModularSalt("$6$", rawSalt, 16)
		res, err := sha512_crypt.New().Generate([]byte(password), []byte(salt))
		if err != nil {
			if jsonMode {
				common.RenderError("cryptpw", 1, "HASH_FAILED", err.Error(), true, stderr)
			} else {
				fmt.Fprintf(stderr, "cryptpw: sha512 hashing failed: %v\n", err)
			}
			return 1
		}
		hash = res

	default:
		if jsonMode {
			common.RenderError("cryptpw", 1, "INVALID_METHOD", "unknown method: "+method, true, stderr)
		} else {
			fmt.Fprintf(stderr, "cryptpw: unknown method: %s\n", method)
		}
		return 1
	}

	if jsonMode {
		common.Render("cryptpw", CryptpwResult{
			Password: password,
			Method:   method,
			Salt:     rawSalt,
			Hash:     hash,
		}, true, stdout, nil)
	} else {
		fmt.Fprintln(stdout, hash)
	}

	return 0
}

func buildModularSalt(prefix, rawSalt string, maxSaltLen int) string {
	if strings.HasPrefix(rawSalt, prefix) {
		return rawSalt
	}
	parts := strings.Split(rawSalt, "$")
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "rounds=") {
		saltPart := parts[1]
		if len(saltPart) > maxSaltLen {
			saltPart = saltPart[:maxSaltLen]
		}
		return prefix + parts[0] + "$" + saltPart
	}

	saltPart := rawSalt
	if len(saltPart) > maxSaltLen {
		saltPart = saltPart[:maxSaltLen]
	}
	return prefix + saltPart
}
