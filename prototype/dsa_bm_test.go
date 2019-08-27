package prototype

import (
	crypto2 "crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"golang.org/x/crypto/ed25519"
	"math/big"
	rand2 "math/rand"
	"testing"
	"time"
)

func newRandString(size int) string {
	r := make([]byte, size)
	_, _ = rand.Reader.Read(r)
	for i := range r {
		r[i] = 'a' + r[i] % 26
	}
	return string(r)
}

func newRandTrxWithoutSignature() *SignedTransaction {
	op := &TransferOperation{
		From: NewAccountName(newRandString(8)),
		To: NewAccountName(newRandString(8)),
		Amount: NewCoin(uint64(rand2.Intn(10000))),
		Memo: newRandString(20),
	}
	return &SignedTransaction{
		Trx: &Transaction{
			RefBlockNum: 1,
			RefBlockPrefix: 0,
			Expiration: NewTimePointSec(uint32(rand2.Intn(10000))),
			Operations: []*Operation { GetPbOperation(op) },
		},
	}
}

func bmTrxSign(b *testing.B) {
	b.StopTimer()
	sk, _ := GenerateNewKey()
	trxs := make([]*SignedTransaction, b.N)
	for i := 0; i < b.N; i++ {
		trxs[i] = newRandTrxWithoutSignature()
	}
	chain := ChainId{Value: 0}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		if sig := trxs[i].Sign(sk, chain); len(sig) > 0 {
			trxs[i].Signature = &SignatureType{Sig: sig}
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func bmTrxVerify(b *testing.B) {
	b.StopTimer()
	sk, _ := GenerateNewKey()
	pk, _ := sk.PubKey()
	trxs := make([]*SignedTransaction, b.N)
	for i := 0; i < b.N; i++ {
		trxs[i] = newRandTrxWithoutSignature()
	}
	chain := ChainId{Value: 0}
	for i := 0; i < b.N; i++ {
		if sig := trxs[i].Sign(sk, chain); len(sig) > 0 {
			trxs[i].Signature = &SignatureType{Sig: sig}
		}
	}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		if trxs[i].VerifySig(pk, chain) {
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func bmTrxExportPub(b *testing.B) {
	b.StopTimer()
	sk, _ := GenerateNewKey()
	trxs := make([]*SignedTransaction, b.N)
	for i := 0; i < b.N; i++ {
		trxs[i] = newRandTrxWithoutSignature()
	}
	chain := ChainId{Value: 0}
	for i := 0; i < b.N; i++ {
		if sig := trxs[i].Sign(sk, chain); len(sig) > 0 {
			trxs[i].Signature = &SignatureType{Sig: sig}
		}
	}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		if _, err := trxs[i].ExportPubKeys(chain); err == nil {
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func bmSecp256k1Sign(b *testing.B) {
	b.StopTimer()
	sk, _ := crypto.GenerateKey()
	msgs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		msgs[i] = []byte(newRandString(256))
	}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		h := sha256.Sum256(msgs[i])
		if _, err := crypto.Sign(h[:], sk); err == nil {
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func bmSecp256k1Verify(b *testing.B) {
	b.StopTimer()
	sk, _ := crypto.GenerateKey()
	pk := crypto.CompressPubkey(&sk.PublicKey)
	msgs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		msgs[i] = []byte(newRandString(256))
	}
	sigs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		h := sha256.Sum256(msgs[i])
		sigs[i], _ = crypto.Sign(h[:], sk)
	}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		h := sha256.Sum256(msgs[i])
		if secp256k1.VerifySignature(pk, h[:], sigs[i][:64]) {
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func bmSecp256k1RecPub(b *testing.B) {
	b.StopTimer()
	sk, _ := crypto.GenerateKey()
	msgs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		msgs[i] = []byte(newRandString(256))
	}
	sigs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		h := sha256.Sum256(msgs[i])
		sigs[i], _ = crypto.Sign(h[:], sk)
	}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		h := sha256.Sum256(msgs[i])
		if _, err := secp256k1.RecoverPubkey(h[:], sigs[i]); err == nil {
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func bmEd25519Sign(b *testing.B) {
	b.StopTimer()
	_, sk, _ := ed25519.GenerateKey(rand.Reader)
	msgs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		msgs[i] = []byte(newRandString(256))
	}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		if sig := ed25519.Sign(sk, msgs[i]); len(sig) > 0 {
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func bmEd25519Verify(b *testing.B) {
	b.StopTimer()
	pk, sk, _ := ed25519.GenerateKey(rand.Reader)
	msgs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		msgs[i] = []byte(newRandString(256))
	}
	sigs := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		sigs[i] = ed25519.Sign(sk, msgs[i])
	}
	b.StartTimer()
	counter := 0
	for i := 0; i < b.N; i++ {
		if ed25519.Verify(pk, msgs[i], sigs[i]) {
			counter++
		}
	}
	if counter != b.N {
		b.Fatalf("%d/%d", counter, b.N)
	}
}

func makeBenchmarkECCurveSign(curve elliptic.Curve) func(*testing.B) {
	return func(b *testing.B) {
		b.StopTimer()
		sk, _ := ecdsa.GenerateKey(curve, rand.Reader)
		msgs := make([][]byte, b.N)
		for i := 0; i < b.N; i++ {
			msgs[i] = []byte(newRandString(256))
		}
		b.StartTimer()
		counter := 0
		for i := 0; i < b.N; i++ {
			h := sha256.Sum256(msgs[i])
			if _, _, err := ecdsa.Sign(rand.Reader, sk, h[:]); err == nil {
				counter++
			}
		}
		if counter != b.N {
			b.Fatalf("%d/%d", counter, b.N)
		}
	}
}

func makeBenchmarkECCurveVerify(curve elliptic.Curve) func(*testing.B) {
	return func(b *testing.B) {
		b.StopTimer()
		sk, _ := ecdsa.GenerateKey(curve, rand.Reader)
		msgs := make([][]byte, b.N)
		for i := 0; i < b.N; i++ {
			msgs[i] = []byte(newRandString(256))
		}
		r := make([]*big.Int, b.N)
		s := make([]*big.Int, b.N)
		for i := 0; i < b.N; i++ {
			h := sha256.Sum256(msgs[i])
			r[i], s[i], _ = ecdsa.Sign(rand.Reader, sk, h[:])
		}
		b.StartTimer()
		counter := 0
		for i := 0; i < b.N; i++ {
			h := sha256.Sum256(msgs[i])
			if ecdsa.Verify(&sk.PublicKey, h[:], r[i], s[i]) {
				counter++
			}
		}
		if counter != b.N {
			b.Fatalf("%d/%d", counter, b.N)
		}
	}
}

func makeBenchmarkECCurve(curve elliptic.Curve) func(*testing.B) {
	return func(b *testing.B) {
		b.Run("sign", makeBenchmarkECCurveSign(curve))
		b.Run("verify", makeBenchmarkECCurveVerify(curve))
	}
}

func makeBenchmarkRSASign(bits int) func(*testing.B) {
	return func(b *testing.B) {
		b.StopTimer()
		sk, _ := rsa.GenerateKey(rand.Reader, bits)
		msgs := make([][]byte, b.N)
		for i := 0; i < b.N; i++ {
			msgs[i] = []byte(newRandString(256))
		}
		b.StartTimer()
		counter := 0
		for i := 0; i < b.N; i++ {
			h := sha256.Sum256(msgs[i])
			if _, err := rsa.SignPKCS1v15(rand.Reader, sk, crypto2.SHA256, h[:]); err == nil {
				counter++
			}
		}
		if counter != b.N {
			b.Fatalf("%d/%d", counter, b.N)
		}
	}
}

func makeBenchmarkRSAVerify(bits int) func(*testing.B) {
	return func(b *testing.B) {
		b.StopTimer()
		sk, _ := rsa.GenerateKey(rand.Reader, bits)
		msgs := make([][]byte, b.N)
		for i := 0; i < b.N; i++ {
			msgs[i] = []byte(newRandString(256))
		}
		sigs := make([][]byte, b.N)
		for i := 0; i < b.N; i++ {
			h := sha256.Sum256(msgs[i])
			sigs[i], _ = rsa.SignPKCS1v15(rand.Reader, sk, crypto2.SHA256, h[:])
		}
		b.StartTimer()
		counter := 0
		for i := 0; i < b.N; i++ {
			h := sha256.Sum256(msgs[i])
			if err := rsa.VerifyPKCS1v15(&sk.PublicKey, crypto2.SHA256, h[:], sigs[i]); err == nil {
				counter++
			}
		}
		if counter != b.N {
			b.Fatalf("%d/%d", counter, b.N)
		}
	}
}

func makeBenchmarkRSAWithBits(bits int) func(*testing.B) {
	return func(b *testing.B) {
		b.Run("sign", makeBenchmarkRSASign(bits))
		b.Run("verify", makeBenchmarkRSAVerify(bits))
	}
}

func benchmarkTrxDSA(b *testing.B) {
	b.Run("sign", bmTrxSign)
	b.Run("verify", bmTrxVerify)
	b.Run("export", bmTrxExportPub)
}

func benchmarkSecp256k1(b *testing.B) {
	b.Run("sign", bmSecp256k1Sign)
	b.Run("verify", bmSecp256k1Verify)
	b.Run("recover", bmSecp256k1RecPub)
}

func benchmarkEcdsa(b *testing.B) {
	// curve: S256, i.e. secp256k1, provided by secp256k1 package
	b.Run("S256", makeBenchmarkECCurve(secp256k1.S256()))
	// curve: P224
	b.Run("P224", makeBenchmarkECCurve(elliptic.P224()))
	// curve: P256
	b.Run("P256", makeBenchmarkECCurve(elliptic.P256()))
	// curve: P384
	b.Run("P384", makeBenchmarkECCurve(elliptic.P384()))
	// curve: P521
	b.Run("P521", makeBenchmarkECCurve(elliptic.P521()))
}

func benchmarkEd25519(b *testing.B) {
	b.Run("sign", bmEd25519Sign)
	b.Run("verify", bmEd25519Verify)
}

func benchmarkRSA(b *testing.B) {
	// rsa 1024bit
	b.Run("1024", makeBenchmarkRSAWithBits(1024))
	// rsa 2048bit
	b.Run("2048", makeBenchmarkRSAWithBits(2048))
	// rsa 3072bit
	b.Run("3072", makeBenchmarkRSAWithBits(3072))
}

func BenchmarkDSA(b *testing.B) {
	rand2.Seed(time.Now().UnixNano())

	// benchmark SignedTransaction.Sign/VerifySig/ExportPubKeys
	b.Run("trx", benchmarkTrxDSA)

	// benchmark contentos-go/common/crypto/secp256k1
	b.Run("s256k1", benchmarkSecp256k1)

	// benchmark crypto/ecdsa
	b.Run("ecdsa", benchmarkEcdsa)

	// benchmark golang.org/x/crypto/ed25519
	b.Run("ed25519", benchmarkEd25519)

	// benchmark crypto/rsa
	b.Run("rsa", benchmarkRSA)
}
