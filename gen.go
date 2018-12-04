package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"strconv"
	"strings"
	"text/template"
)

func main() {
	var (
		fname   = flag.String("o", "", "output file (defaults to stdout)")
		pkgname = flag.String("p", "main", "output package name")
	)
	flag.Parse()

	specs := flag.Args()
	if len(specs) < 1 {
		fmt.Println("must specifiy at least one key spec")
		os.Exit(1)
	}

	var out io.Writer = os.Stdout
	if len(*fname) > 0 {
		f, err := os.OpenFile(*fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		out = f
	}

	keys := make([]struct {
		Name  string
		Block string
	}, len(flag.Args()))

	buf := &bytes.Buffer{}
	for i, arg := range flag.Args() {
		name, data, err := generate(arg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		buf.Reset()
		err = pem.Encode(buf, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: data,
		})

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		keys[i].Name = name
		keys[i].Block = buf.String()
	}

	buf.Reset()
	err := tmpl.Execute(buf, struct {
		Package string
		Keys    interface{}
	}{
		Package: *pkgname,
		Keys:    keys,
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	data, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = out.Write(data)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func generate(spec string) (string, []byte, error) {
	parts := strings.Split(spec, ":")
	if len(parts) != 3 {
		return "", []byte{}, fmt.Errorf("invalid key spec '%s'", spec)
	}

	switch parts[1] {
	case "rsa":
		size, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return "", []byte{}, fmt.Errorf("invalid key size: %v", err)
		}
		key, err := rsa.GenerateKey(rand.Reader, int(size))
		if err != nil {
			return "", []byte{}, fmt.Errorf("failed to generate key: %v", err)
		}
		return parts[0], x509.MarshalPKCS1PrivateKey(key), nil
	default:
		return "", []byte{}, fmt.Errorf("unsupported key type '%s'", parts[1])
	}

}

var tmpl = template.Must(template.New("keys").Parse(`
// Generated by go-test-keys. DO NOT EDIT. 

package {{.Package}}

import (
    "crypto/x509"
    "crypto/rsa"
    "encoding/pem"
)

var (
    {{- range .Keys}}
    {{.Name}} *rsa.PrivateKey
    {{- end}}
)

func init() {
    parse := func(enc string) *rsa.PrivateKey {
        block, _ := pem.Decode([]byte(enc))
        key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
        return key
    }

    {{range .Keys}}
    {{.Name}} = parse(` + "`\n{{.Block}}`" + `)
    {{end}}
}
`))
