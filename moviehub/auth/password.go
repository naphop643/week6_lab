package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword แปลงรหัสผ่านเป็น bcrypt hash ที่ย้อนกลับเป็นรหัสผ่านเดิมไม่ได้
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword(
		[]byte(plain),
		bcrypt.DefaultCost,
	)
	return string(b), err
}

// CheckPassword ตรวจว่ารหัสผ่านตรงกับ hash ที่บันทึกไว้หรือไม่
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(plain),
	) == nil
}
