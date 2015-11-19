// Package gpgme provides a Go wrapper for the GPGME library
package gpgme

// #cgo LDFLAGS: -lgpgme -lassuan -lgpg-error
// #include <stdlib.h>
// #include <string.h>
// #include <gpgme.h>
// #include <errno.h>
// #include "go_gpgme.h"
import "C"

import (
	"io"
	"os"
	"runtime"
	"time"
	"unsafe"
)

var Version string

func init() {
	Version = C.GoString(C.gpgme_check_version(nil))
}

//export gogpgme_readfunc
func gogpgme_readfunc(handle, buffer unsafe.Pointer, size C.size_t) C.ssize_t {
	d := (*Data)(handle)
	buf := make([]byte, size)
	n, err := d.r.Read(buf)
	if err != nil && err != io.EOF {
		C.gpgme_err_set_errno(C.EIO)
		return -1
	}
	C.memcpy(buffer, unsafe.Pointer(&buf[0]), C.size_t(n))
	return C.ssize_t(n)
}

//export gogpgme_writefunc
func gogpgme_writefunc(handle, buffer unsafe.Pointer, size C.size_t) C.ssize_t {
	d := (*Data)(handle)
	n, err := d.w.Write(C.GoBytes(buffer, C.int(size)))
	if err != nil && err != io.EOF {
		C.gpgme_err_set_errno(C.EIO)
		return -1
	}
	return C.ssize_t(n)
}

//export gogpgme_seekfunc
func gogpgme_seekfunc(handle unsafe.Pointer, offset C.off_t, whence C.int) C.off_t {
	d := (*Data)(handle)
	n, err := d.s.Seek(int64(offset), int(whence))
	if err != nil {
		C.gpgme_err_set_errno(C.EIO)
		return -1
	}
	return C.off_t(n)
}

// Callback is the function that is called when a passphrase is required
type Callback func(uidHint string, prevWasBad bool, f *os.File) error

//export gogpgme_passfunc
func gogpgme_passfunc(hook unsafe.Pointer, uid_hint, passphrase_info *C.char, prev_was_bad, fd C.int) C.gpgme_error_t {
	c := (*Context)(hook)
	go_uid_hint := C.GoString(uid_hint)
	f := os.NewFile(uintptr(fd), go_uid_hint)
	defer f.Close()
	err := c.callback(go_uid_hint, prev_was_bad != 0, f)
	if err != nil {
		return C.GPG_ERR_CANCELED
	}
	return 0
}

type Protocol int

const (
	ProtocolOpenPGP  Protocol = C.GPGME_PROTOCOL_OpenPGP
	ProtocolCMS               = C.GPGME_PROTOCOL_CMS
	ProtocolGPGConf           = C.GPGME_PROTOCOL_GPGCONF
	ProtocolAssuan            = C.GPGME_PROTOCOL_ASSUAN
	ProtocolG13               = C.GPGME_PROTOCOL_G13
	ProtocolUIServer          = C.GPGME_PROTOCOL_UISERVER
	ProtocolSpawn             = C.GPGME_PROTOCOL_SPAWN
	ProtocolDefault           = C.GPGME_PROTOCOL_DEFAULT
	ProtocolUnknown           = C.GPGME_PROTOCOL_UNKNOWN
)

type PinEntryMode int

const (
	PinEntryDefault  PinEntryMode = C.GPGME_PINENTRY_MODE_DEFAULT
	PinEntryAsk                   = C.GPGME_PINENTRY_MODE_ASK
	PinEntryCancel                = C.GPGME_PINENTRY_MODE_CANCEL
	PinEntryError                 = C.GPGME_PINENTRY_MODE_ERROR
	PinEntryLoopback              = C.GPGME_PINENTRY_MODE_LOOPBACK
)

type EncryptFlag uint

const (
	EncryptAlwaysTrust EncryptFlag = C.GPGME_ENCRYPT_ALWAYS_TRUST
	EncryptNoEncryptTo             = C.GPGME_ENCRYPT_NO_ENCRYPT_TO
	EncryptPrepare                 = C.GPGME_ENCRYPT_PREPARE
	EncryptExceptSign              = C.GPGME_ENCRYPT_EXPECT_SIGN
	EncryptNoCompress              = C.GPGME_ENCRYPT_NO_COMPRESS
)

type KeyListMode uint

const (
	KeyListModeLocal        KeyListMode = C.GPGME_KEYLIST_MODE_LOCAL
	KeyListModeExtern                   = C.GPGME_KEYLIST_MODE_EXTERN
	KeyListModeSigs                     = C.GPGME_KEYLIST_MODE_SIGS
	KeyListModeSigNotations             = C.GPGME_KEYLIST_MODE_SIG_NOTATIONS
	KeyListModeWithSecret               = C.GPGME_KEYLIST_MODE_WITH_SECRET
	KeyListModeEphemeral                = C.GPGME_KEYLIST_MODE_EPHEMERAL
	KeyListModeModeValidate             = C.GPGME_KEYLIST_MODE_VALIDATE
)

type Validity int

const (
	ValidityUnknown   Validity = C.GPGME_VALIDITY_UNKNOWN
	ValidityUndefined          = C.GPGME_VALIDITY_UNDEFINED
	ValidityNever              = C.GPGME_VALIDITY_NEVER
	ValidityMarginal           = C.GPGME_VALIDITY_MARGINAL
	ValidityFull               = C.GPGME_VALIDITY_FULL
	ValidityUltimate           = C.GPGME_VALIDITY_ULTIMATE
)

type ErrorCode int

const (
	ErrorNoError ErrorCode = C.GPG_ERR_NO_ERROR
	ErrorEOF               = C.GPG_ERR_EOF
)

// Error is a wrapper for GPGME errors
type Error struct {
	err C.gpgme_error_t
}

func (e Error) Code() ErrorCode {
	return ErrorCode(C.gpgme_err_code(e.err))
}

func (e Error) Error() string {
	return C.GoString(C.gpgme_strerror(e.err))
}

func handleError(err C.gpgme_error_t) error {
	e := Error{err: err}
	if e.Code() == ErrorNoError {
		return nil
	}
	return e
}

func handleErrno(err error) error {
	return err
}

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func EngineCheckVersion(p Protocol) error {
	return handleError(C.gpgme_engine_check_version(C.gpgme_protocol_t(p)))
}

type EngineInfo struct {
	info C.gpgme_engine_info_t
}

func (e *EngineInfo) Next() *EngineInfo {
	if e.info.next == nil {
		return nil
	}
	return &EngineInfo{info: e.info.next}
}

func (e *EngineInfo) Protocol() Protocol {
	return Protocol(e.info.protocol)
}

func (e *EngineInfo) FileName() string {
	return C.GoString(e.info.file_name)
}

func (e *EngineInfo) Version() string {
	return C.GoString(e.info.version)
}

func (e *EngineInfo) RequiredVersion() string {
	return C.GoString(e.info.req_version)
}

func (e *EngineInfo) HomeDir() string {
	return C.GoString(e.info.home_dir)
}

func GetEngineInfo() (*EngineInfo, error) {
	info := &EngineInfo{}
	return info, handleError(C.gpgme_get_engine_info(&info.info))
}

func FindKeys(pattern string, secretOnly bool) ([]*Key, error) {
	var keys []*Key
	ctx, err := New()
	if err != nil {
		return keys, err
	}
	defer ctx.Release()
	if err := ctx.KeyListStart(pattern, secretOnly); err != nil {
		return keys, err
	}
	defer ctx.KeyListEnd()
	for ctx.KeyListNext() {
		keys = append(keys, ctx.Key)
	}
	if ctx.KeyError != nil {
		return keys, ctx.KeyError
	}
	return keys, nil
}

func Decrypt(r io.Reader) (*Data, error) {
	ctx, err := New()
	if err != nil {
		return nil, err
	}
	defer ctx.Release()
	cipher, err := NewDataReader(r)
	if err != nil {
		return nil, err
	}
	defer cipher.Close()
	plain, err := NewData()
	if err != nil {
		return nil, err
	}
	err = ctx.Decrypt(cipher, plain)
	plain.Seek(0, 0)
	return plain, err
}

type Context struct {
	Key      *Key
	KeyError error

	callback Callback

	ctx C.gpgme_ctx_t
}

func New() (*Context, error) {
	c := &Context{}
	err := C.gpgme_new(&c.ctx)
	runtime.SetFinalizer(c, (*Context).Release)
	return c, handleError(err)
}

func (c *Context) Release() {
	if c.ctx == nil {
		return
	}
	C.gpgme_release(c.ctx)
	c.ctx = nil
}

func (c *Context) SetArmor(yes bool) {
	C.gpgme_set_armor(c.ctx, cbool(yes))
}

func (c *Context) Armor() bool {
	return C.gpgme_get_armor(c.ctx) != 0
}

func (c *Context) SetProtocol(p Protocol) error {
	return handleError(C.gpgme_set_protocol(c.ctx, C.gpgme_protocol_t(p)))
}

func (c *Context) Protocol() Protocol {
	return Protocol(C.gpgme_get_protocol(c.ctx))
}

func (c *Context) SetKeyListMode(m KeyListMode) error {
	return handleError(C.gpgme_set_keylist_mode(c.ctx, C.gpgme_keylist_mode_t(m)))
}

func (c *Context) KeyListMode() KeyListMode {
	return KeyListMode(C.gpgme_get_keylist_mode(c.ctx))
}

func (c *Context) SetPinEntryMode(m PinEntryMode) error {
	return handleError(C.gpgme_set_pinentry_mode(c.ctx, C.gpgme_pinentry_mode_t(m)))
}

func (c *Context) PinEntryMode() PinEntryMode {
	return PinEntryMode(C.gpgme_get_pinentry_mode(c.ctx))
}

func (c *Context) SetCallback(callback Callback) error {
	var err error
	c.callback = callback
	if callback != nil {
		_, err = C.gpgme_set_passphrase_cb(c.ctx, C.gpgme_passphrase_cb_t(C.gogpgme_passfunc), unsafe.Pointer(c))
	} else {
		_, err = C.gpgme_set_passphrase_cb(c.ctx, nil, nil)
	}
	return err
}

func (c *Context) EngineInfo() *EngineInfo {
	return &EngineInfo{info: C.gpgme_ctx_get_engine_info(c.ctx)}
}

func (c *Context) KeyListStart(pattern string, secretOnly bool) error {
	cpattern := C.CString(pattern)
	defer C.free(unsafe.Pointer(cpattern))
	err := C.gpgme_op_keylist_start(c.ctx, cpattern, cbool(secretOnly))
	return handleError(err)
}

func (c *Context) KeyListNext() bool {
	c.Key = newKey()
	err := handleError(C.gpgme_op_keylist_next(c.ctx, &c.Key.k))
	if err != nil {
		if e, ok := err.(Error); ok && e.Code() == ErrorEOF {
			c.KeyError = nil
		} else {
			c.KeyError = err
		}
		return false
	}
	c.KeyError = nil
	return true
}

func (c *Context) KeyListEnd() error {
	return handleError(C.gpgme_op_keylist_end(c.ctx))
}

func (c *Context) Decrypt(ciphertext, plaintext *Data) error {
	return handleError(C.gpgme_op_decrypt(c.ctx, ciphertext.dh, plaintext.dh))
}

func (c *Context) DecryptVerify(ciphertext, plaintext *Data) error {
	return handleError(C.gpgme_op_decrypt_verify(c.ctx, ciphertext.dh, plaintext.dh))
}

func (c *Context) Encrypt(recipients []*Key, flags EncryptFlag, plaintext, ciphertext *Data) error {
	size := unsafe.Sizeof(new(C.gpgme_key_t))
	recp := C.calloc(C.size_t(len(recipients)+1), C.size_t(size))
	defer C.free(recp)
	for i := range recipients {
		ptr := (*C.gpgme_key_t)(unsafe.Pointer(uintptr(recp) + size*uintptr(i)))
		*ptr = recipients[i].k
	}
	err := C.gpgme_op_encrypt(c.ctx, (*C.gpgme_key_t)(recp), C.gpgme_encrypt_flags_t(flags), plaintext.dh, ciphertext.dh)
	return handleError(err)
}

type Key struct {
	k C.gpgme_key_t
}

func newKey() *Key {
	k := &Key{}
	runtime.SetFinalizer(k, (*Key).Release)
	return k
}

func (k *Key) Release() {
	C.gpgme_key_release(k.k)
	k.k = nil
}

func (k *Key) Revoked() bool {
	return C.key_revoked(k.k) != 0
}

func (k *Key) Expired() bool {
	return C.key_expired(k.k) != 0
}

func (k *Key) Disabled() bool {
	return C.key_disabled(k.k) != 0
}

func (k *Key) Invalid() bool {
	return C.key_invalid(k.k) != 0
}

func (k *Key) CanEncrypt() bool {
	return C.key_can_encrypt(k.k) != 0
}

func (k *Key) CanSign() bool {
	return C.key_can_sign(k.k) != 0
}

func (k *Key) CanCertify() bool {
	return C.key_can_certify(k.k) != 0
}

func (k *Key) Secret() bool {
	return C.key_secret(k.k) != 0
}

func (k *Key) CanAuthenticate() bool {
	return C.key_can_authenticate(k.k) != 0
}

func (k *Key) IsQualified() bool {
	return C.key_is_qualified(k.k) != 0
}

func (k *Key) Protocol() Protocol {
	return Protocol(k.k.protocol)
}

func (k *Key) IssuerSerial() string {
	return C.GoString(k.k.issuer_serial)
}

func (k *Key) IssuerName() string {
	return C.GoString(k.k.issuer_name)
}

func (k *Key) ChainID() string {
	return C.GoString(k.k.chain_id)
}

func (k *Key) OwnerTrust() Validity {
	return Validity(k.k.owner_trust)
}

func (k *Key) SubKeys() *SubKey {
	if k.k.subkeys == nil {
		return nil
	}
	return &SubKey{k: k.k.subkeys, parent: k}
}

func (k *Key) UserIDs() *UserID {
	if k.k.uids == nil {
		return nil
	}
	return &UserID{u: k.k.uids, parent: k}
}

func (k *Key) KeyListMode() KeyListMode {
	return KeyListMode(k.k.keylist_mode)
}

type SubKey struct {
	k      C.gpgme_subkey_t
	parent *Key // make sure the key is not released when we have a reference to a subkey
}

func (k *SubKey) Next() *SubKey {
	if k.k.next == nil {
		return nil
	}
	return &SubKey{k: k.k.next, parent: k.parent}
}

func (k *SubKey) Revoked() bool {
	return C.subkey_revoked(k.k) != 0
}

func (k *SubKey) Expired() bool {
	return C.subkey_expired(k.k) != 0
}

func (k *SubKey) Disabled() bool {
	return C.subkey_disabled(k.k) != 0
}

func (k *SubKey) Invalid() bool {
	return C.subkey_invalid(k.k) != 0
}

func (k *SubKey) Secret() bool {
	return C.subkey_secret(k.k) != 0
}

func (k *SubKey) KeyID() string {
	return C.GoString(k.k.keyid)
}

func (k *SubKey) Fingerprint() string {
	return C.GoString(k.k.fpr)
}

func (k *SubKey) Created() time.Time {
	if k.k.timestamp <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(k.k.timestamp), 0)
}

func (k *SubKey) Expires() time.Time {
	if k.k.expires <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(k.k.expires), 0)
}

func (k *SubKey) CardNumber() string {
	return C.GoString(k.k.card_number)
}

type UserID struct {
	u      C.gpgme_user_id_t
	parent *Key // make sure the key is not released when we have a reference to a user ID
}

func (u *UserID) Next() *UserID {
	if u.u.next == nil {
		return nil
	}
	return &UserID{u: u.u.next, parent: u.parent}
}

func (u *UserID) Revoked() bool {
	return C.uid_revoked(u.u) != 0
}

func (u *UserID) Invalid() bool {
	return C.uid_invalid(u.u) != 0
}

func (u *UserID) Validity() Validity {
	return Validity(u.u.validity)
}

func (u *UserID) UID() string {
	return C.GoString(u.u.uid)
}

func (u *UserID) Name() string {
	return C.GoString(u.u.name)
}

func (u *UserID) Comment() string {
	return C.GoString(u.u.comment)
}

func (u *UserID) Email() string {
	return C.GoString(u.u.email)
}

// The Data buffer used to communicate with GPGME
type Data struct {
	dh  C.gpgme_data_t
	cbs C.struct_gpgme_data_cbs
	r   io.Reader
	w   io.Writer
	s   io.Seeker
}

func newData() *Data {
	d := &Data{}
	runtime.SetFinalizer(d, (*Data).Close)
	return d
}

// NewData returns a new memory based data buffer
func NewData() (*Data, error) {
	d := newData()
	return d, handleError(C.gpgme_data_new(&d.dh))
}

// NewDataFile returns a new file based data buffer
func NewDataFile(f *os.File) (*Data, error) {
	d := newData()
	return d, handleError(C.gpgme_data_new_from_fd(&d.dh, C.int(f.Fd())))
}

// NewDataBytes returns a new memory based data buffer that contains `b` bytes
func NewDataBytes(b []byte) (*Data, error) {
	d := newData()
	return d, handleError(C.gpgme_data_new_from_mem(&d.dh, (*C.char)(unsafe.Pointer(&b[0])), C.size_t(len(b)), 1))
}

// NewDataReader returns a new callback based data buffer
func NewDataReader(r io.Reader) (*Data, error) {
	d := newData()
	d.r = r
	d.cbs.read = C.gpgme_data_read_cb_t(C.gogpgme_readfunc)
	return d, handleError(C.gpgme_data_new_from_cbs(&d.dh, &d.cbs, unsafe.Pointer(d)))
}

// NewDataWriter returns a new callback based data buffer
func NewDataWriter(w io.Writer) (*Data, error) {
	d := newData()
	d.w = w
	d.cbs.write = C.gpgme_data_write_cb_t(C.gogpgme_writefunc)
	return d, handleError(C.gpgme_data_new_from_cbs(&d.dh, &d.cbs, unsafe.Pointer(d)))
}

// NewDataReadWriter returns a new callback based data buffer
func NewDataReadWriter(rw io.ReadWriter) (*Data, error) {
	d := newData()
	d.r = rw
	d.w = rw
	d.cbs.read = C.gpgme_data_read_cb_t(C.gogpgme_readfunc)
	d.cbs.write = C.gpgme_data_write_cb_t(C.gogpgme_writefunc)
	return d, handleError(C.gpgme_data_new_from_cbs(&d.dh, &d.cbs, unsafe.Pointer(d)))
}

// NewDataReadWriteSeeker returns a new callback based data buffer
func NewDataReadWriteSeeker(rw io.ReadWriteSeeker) (*Data, error) {
	d := newData()
	d.r = rw
	d.w = rw
	d.s = rw
	d.cbs.read = C.gpgme_data_read_cb_t(C.gogpgme_readfunc)
	d.cbs.write = C.gpgme_data_write_cb_t(C.gogpgme_writefunc)
	d.cbs.seek = C.gpgme_data_seek_cb_t(C.gogpgme_seekfunc)
	return d, handleError(C.gpgme_data_new_from_cbs(&d.dh, &d.cbs, unsafe.Pointer(d)))
}

// Close releases any resources associated with the data buffer
func (d *Data) Close() error {
	if d.dh == nil {
		return nil
	}
	_, err := C.gpgme_data_release(d.dh)
	d.dh = nil
	return err
}

func (d *Data) Write(p []byte) (int, error) {
	n, err := C.gpgme_data_write(d.dh, unsafe.Pointer(&p[0]), C.size_t(len(p)))
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return int(n), nil
}

func (d *Data) Read(p []byte) (int, error) {
	n, err := C.gpgme_data_read(d.dh, unsafe.Pointer(&p[0]), C.size_t(len(p)))
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return int(n), nil
}

func (d *Data) Seek(offset int64, whence int) (int64, error) {
	n, err := C.gpgme_data_seek(d.dh, C.off_t(offset), C.int(whence))
	return int64(n), err
}

// Name returns the associated filename if any
func (d *Data) Name() string {
	return C.GoString(C.gpgme_data_get_file_name(d.dh))
}