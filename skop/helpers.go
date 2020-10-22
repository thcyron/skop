package skop

func Bool(b bool) *bool          { return &b }
func Int(i int) *int             { return &i }
func Int32(i int32) *int32       { return &i }
func Int64(i int64) *int64       { return &i }
func Uint(u uint) *uint          { return &u }
func Uint32(u uint32) *uint32    { return &u }
func Uint64(u uint64) *uint64    { return &u }
func Float32(f float32) *float32 { return &f }
func Float64(f float64) *float64 { return &f }
func String(s string) *string    { return &s }
