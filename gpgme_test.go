package gpgme

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"
)

const (
	testData       = "data\n"
	testCipherText = `-----BEGIN PGP MESSAGE-----
Version: GnuPG v1

hQEMAw4698hF4WUhAQf/SdkbF/zUE6YjBxscDurrUZunSnt87kipLXypSxTDIdgj
O9huAaQwBz4uAJf2DuEN/7iAFGhi/v45NTujrG+7ocfjM3m/A2T80g4RVF5kKXBr
pFFgH7bMRY6VdZt1GKI9izSO/uFkoKXG8M31tCX3hWntQUJ9p+n1avGpu3wo4Ru3
CJhpL+ChDzXuZv4IK81ahrixEz4fJH0vd0TbsHpTXx4WPkTGXelM0R9PwiY7TovZ
onGZUIplrfA1HUUbQfzExyFw3oAo1/almzD5CBIfq5EnU8Siy5BNulDXm0/44h8A
lOUy6xqx7ITnWMYYf4a1cFoW80Yk+x6SYQqbcqHFQdJIAVr00V4pPV4ppFcXgdw/
BxKERzDupyqS0tcfVFCYLRmvtQp7ceOS6jRW3geXPPIz1U/VYBvKlvFu4XTMCS6z
4qY4SzZlFEsU
=IQOA
-----END PGP MESSAGE-----`
	textSignedCipherText = `-----BEGIN PGP MESSAGE-----
Version: GnuPG v1

hQEMAw4698hF4WUhAQf9FaI8zSltIMcDxqGtpcTfcBtC30rcKqE/oa1bacxuZQKW
uT671N7J2TjU8tLIsAO9lDSRHCsf6DbB+O6qOJsu6eJownyMUEmJ2oBON9Z2x+Ci
aS0KeRqcRxNJYbIth9/ffa2I4sSBA0GS93yCfGKHNL4JvNo/dgVjJwOPWZ5TKu49
n9h81LMwTQ8qpLGPeo7nsgBtSKNTi5s7pC0bd9HTii7gQcLUEeeqk5U7oG5CRN48
zKPrVud8pIDvSoYuymZmjWI1MmF4xQ+6IHK7hWKMyA2RkS4CpyBPERMifc+y+K2+
LnKLt2ul9jxgqv3fd/dU7UhyqdvJdj83tUvhN4Iru9LAuQHfVsP23keH8CHpUzCj
PkuUYvpgmRdMSC+l3yBCqnaaAI51rRdyJ+HX4DrrinOFst2JfXEICiQAg5UkO4i/
YYHsfD3xfk7MKMOY/DfV25OyAU6Yeq/5SMisONhtlzcosVU7HULudbIKSG6q/tCx
jRGu+oMIBdiQ+eipnaLVetyGxSEDzxHH367NbxC9ffxK7I+Srmwh+oS/yMNuZA8r
bUp6NbykK8Si7VNgVG1+yOIULLuR0VBO5+e2PKQU8FP0wABTn7h3zE8z4rCrcaZl
43bc/TBUyTdWY79e2P1otaCbhQGAjfF8ClPbAdsFppykf5I2ZDDf8c67d0Nqui74
Yry4/fF/2sAzyoyzdP6ktBw6pCA1MbwDqTGg7aEI8Es3X+jfh3qIaLnGmZ7jIyUl
D4jgGAEpFU+mpPH8eukKuT+OJ3P6gdmSiAJH7+C96tlcEg9BNxjYNkCTil/3yygQ
EV/5zFTEpzm4CtYHHdmY5uCaEJq/4hhE8BY8
=8mxI
-----END PGP MESSAGE-----`
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Setenv("GNUPGHOME", "./testdata/gpghome")
	os.Exit(m.Run())
}

func TestContext_Armor(t *testing.T) {
	ctx, err := New()
	checkError(t, err)

	ctx.SetArmor(true)
	if !ctx.Armor() {
		t.Error("expected armor set")
	}
	ctx.SetArmor(false)
	if ctx.Armor() {
		t.Error("expected armor not set")
	}
}

func TestContext_Encrypt(t *testing.T) {
	ctx, err := New()
	checkError(t, err)

	keys, err := FindKeys("test@example.com", true)
	checkError(t, err)

	plain, err := NewDataBytes([]byte(testData))
	checkError(t, err)

	var buf bytes.Buffer
	cipher, err := NewDataWriter(&buf)
	checkError(t, err)

	checkError(t, ctx.Encrypt(keys, 0, plain, cipher))
	if buf.Len() < 1 {
		t.Error("Expected encrypted bytes, got empty buffer")
	}
}

func TestContext_Decrypt(t *testing.T) {
	skipGPG2x(t, "can only set password callback for GPG v1.x")

	ctx, err := New()
	checkError(t, err)

	checkError(t, ctx.SetCallback(func(uid_hint string, prev_was_bad bool, f *os.File) error {
		if prev_was_bad {
			t.Fatal("Bad passphrase")
		}
		_, err := io.WriteString(f, "password\n")
		return err
	}))
	cipher, err := NewDataBytes([]byte(testCipherText))
	checkError(t, err)
	var buf bytes.Buffer
	plain, err := NewDataWriter(&buf)
	checkError(t, err)
	checkError(t, ctx.Decrypt(cipher, plain))
	diff(t, buf.Bytes(), []byte("Test message\n"))
}

func TestContext_DecryptVerify(t *testing.T) {
	skipGPG2x(t, "can only set password callback for GPG v1.x")

	ctx, err := New()
	checkError(t, err)

	checkError(t, ctx.SetCallback(func(uid_hint string, prev_was_bad bool, f *os.File) error {
		if prev_was_bad {
			t.Fatal("Bad passphrase")
		}
		_, err := io.WriteString(f, "password\n")
		return err
	}))
	cipher, err := NewDataBytes([]byte(textSignedCipherText))
	checkError(t, err)
	var buf bytes.Buffer
	plain, err := NewDataWriter(&buf)
	checkError(t, err)
	checkError(t, ctx.DecryptVerify(cipher, plain))
	diff(t, buf.Bytes(), []byte("Test message\n"))
}

func TestNewData(t *testing.T) {
	dh, err := NewData()
	checkError(t, err)
	for i := 0; i < 5; i++ {
		_, err := dh.Write([]byte(testData))
		checkError(t, err)
	}
	_, err = dh.Seek(0, 0)
	checkError(t, err)

	var buf bytes.Buffer
	_, err = io.Copy(&buf, dh)
	checkError(t, err)
	expected := bytes.Repeat([]byte(testData), 5)
	diff(t, buf.Bytes(), expected)

	dh.Close()
}

func TestDataNewDataFile(t *testing.T) {
	f, err := ioutil.TempFile("", "gpgme")
	checkError(t, err)
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	dh, err := NewDataFile(f)
	checkError(t, err)
	defer dh.Close()
	for i := 0; i < 5; i++ {
		_, err := dh.Write([]byte(testData))
		checkError(t, err)
	}
	_, err = dh.Seek(0, 0)
	checkError(t, err)
	var buf bytes.Buffer
	_, err = io.Copy(&buf, dh)
	checkError(t, err)
	expected := bytes.Repeat([]byte(testData), 5)
	diff(t, buf.Bytes(), expected)
}

func TestDataNewDataReader(t *testing.T) {
	r := bytes.NewReader([]byte(testData))
	dh, err := NewDataReader(r)
	checkError(t, err)
	var buf bytes.Buffer
	_, err = io.Copy(&buf, dh)
	checkError(t, err)
	diff(t, buf.Bytes(), []byte(testData))

	dh.Close()
}

func TestDataNewDataWriter(t *testing.T) {
	var buf bytes.Buffer
	dh, err := NewDataWriter(&buf)
	checkError(t, err)
	for i := 0; i < 5; i++ {
		_, err := dh.Write([]byte(testData))
		checkError(t, err)
	}
	expected := bytes.Repeat([]byte(testData), 5)
	diff(t, buf.Bytes(), expected)

	dh.Close()
}

func skipGPG2x(t *testing.T, msg string) {
	var info *EngineInfo
	info, err := GetEngineInfo()
	checkError(t, err)
	for info != nil {
		if strings.HasSuffix(info.FileName(), "gpg") && strings.HasPrefix(info.Version(), "2.") {
			t.Skip(msg)
		}
		info = info.Next()
	}

}

func diff(t *testing.T, dst, src []byte) {
	line := 1
	offs := 0 // line offset
	for i := 0; i < len(dst) && i < len(src); i++ {
		d := dst[i]
		s := src[i]
		if d != s {
			t.Errorf("dst:%d: %s\n", line, dst[offs:i+1])
			t.Errorf("src:%d: %s\n", line, src[offs:i+1])
			return
		}
		if s == '\n' {
			line++
			offs = i + 1
		}
	}
	if len(dst) != len(src) {
		t.Errorf("len(dst) = %d, len(src) = %d\nsrc = %q", len(dst), len(src), src)
	}
}

func checkError(t *testing.T, err error) {
	if err == nil {
		return
	}
	_, file, line, ok := runtime.Caller(1)
	if ok {
		// Truncate file name at last file name separator.
		if index := strings.LastIndex(file, "/"); index >= 0 {
			file = file[index+1:]
		} else if index = strings.LastIndex(file, "\\"); index >= 0 {
			file = file[index+1:]
		}
	} else {
		file = "???"
		line = 1
	}
	t.Fatalf("%s:%d: %s", file, line, err)
}