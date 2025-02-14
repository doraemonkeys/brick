package brick

import (
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func Test1(t *testing.T) {
	out, err := exec.Command("go", "test", "github.com/doraemonkeys/brick/test/test1").CombinedOutput()
	if err != nil {
		t.Fatal(string(out))
	}
	out, err = exec.Command("go", "test", "github.com/doraemonkeys/brick/test/test2").CombinedOutput()
	if err != nil {
		t.Fatal(string(out))
	}
	out, err = exec.Command("go", "test", "github.com/doraemonkeys/brick/test/test3").CombinedOutput()
	if err != nil {
		t.Fatal(string(out))
	}
	out, err = exec.Command("go", "test", "github.com/doraemonkeys/brick/test/test4").CombinedOutput()
	if err != nil {
		t.Fatal(string(out))
	}
	out, err = exec.Command("go", "test", "github.com/doraemonkeys/brick/test/test5").CombinedOutput()
	if err != nil {
		t.Fatal(string(out))
	}
	out, err = exec.Command("go", "test", "github.com/doraemonkeys/brick/test/test6").CombinedOutput()
	if err != nil {
		t.Fatal(string(out))
	}
}

type TestBrick1 struct {
	Test string
}

func (t *TestBrick1) BrickTypeID() string {
	return "TestBrick1"
}

type TestBrick2 struct {
	Test string
}

func (t TestBrick2) BrickTypeID() string {
	return "TestBrick2"
}

func TestBrickManager_getTypeIDByReflectType(t *testing.T) {
	tests := []struct {
		name string
		b    *BrickManager
		typ  reflect.Type
		want string
	}{
		{
			name: "1",
			b:    nil,
			typ:  reflect.TypeOf(TestBrick1{}),
			want: "TestBrick1",
		},
		{
			name: "2",
			b:    nil,
			typ:  reflect.TypeOf(TestBrick2{}),
			want: "TestBrick2",
		},
		{
			name: "3",
			b:    nil,
			typ:  reflect.TypeOf(&TestBrick1{}),
			want: "TestBrick1",
		},
		{
			name: "4",
			b:    nil,
			typ:  reflect.TypeOf(&TestBrick2{}),
			want: "TestBrick2",
		}, {
			name: "5",
			b:    nil,
			typ:  reflect.TypeOf((**TestBrick1)(nil)),
			want: "TestBrick1",
		},
		{
			name: "6",
			b:    nil,
			typ:  reflect.TypeOf((**TestBrick2)(nil)),
			want: "TestBrick2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.getTypeIDByReflectType(tt.typ); got != tt.want {
				t.Errorf("BrickManager.getTypeIDByReflectType() = %v, want %v", got, tt.want)
			}
		})
	}
}

type TestBrick3 struct {
	T1 *TestBrick1 `brick:"random"`
}

func (t *TestBrick3) BrickTypeID() string {
	return "TestBrick3"
}

// random liveID test
func TestRandomLiveID(t *testing.T) {
	Register[*TestBrick3]()

	for i := 0; i < 100; i++ {
		instance := GetOrCreate[*TestBrick3](RandomLiveID())
		if instance.T1.Test == "NOT EMPTY" {
			t.Errorf("not random instance")
		}
		instance.T1.Test = "NOT EMPTY"
	}

	t.Run("TestBrick3", func(t *testing.T) {
		defer func() {
			r := recover()
			if r != nil {
				t.Errorf("panic: %v", r)
			}

		}()
		Get[*TestBrick3]()
	})
}

type TestBrick4 struct {
	T1 *TestBrick1 `brick:""`
}

func (t *TestBrick4) BrickTypeID() string {
	return "TestBrick4"
}

func (t *TestBrick4) NewBrick(_ []byte) Brick {
	return &TestBrick4{}
}

func (t *TestBrick4) BrickLives() []Live {
	return []Live{
		{
			LiveID:    "TestBrick4 BrickLives Test",
			RelyLives: map[string]string{"T1": "T1 LIVEID"},
		},
	}
}

func Test_BrickLives(t *testing.T) {
	RegisterLives[*TestBrick4]()

	instance := Get[*TestBrick4]()
	instance.T1.Test = "NOT EMPTY"

	instance2 := GetOrCreate[*TestBrick4]("random liveID")
	if instance2.T1.Test != "NOT EMPTY" {
		t.Errorf("instance2.T1.Test = %v, want %v", instance2.T1.Test, "NOT EMPTY")
	}

	instance3 := Get[*TestBrick4]("TestBrick4 BrickLives Test")
	if instance3.T1.Test == "NOT EMPTY" {
		t.Errorf("instance3.T1.Test = %v, want empty", instance3.T1.Test)
	}
}

type TestBrick5 struct {
	T1  *TestBrick1 `brick:""`
	T12 *TestBrick1 `brick:""`
	T5  *TestBrick5 `brick:""`
}

func (t *TestBrick5) BrickTypeID() string {
	return "TestBrick5"
}

type TestBrick51 struct {
	T1  *TestBrick1  `brick:""`
	T51 *TestBrick51 `brick:"random"`
}

func (t *TestBrick51) BrickTypeID() string {
	return "TestBrick51"
}

type TestBrick52 struct {
	T1  *TestBrick1    `brick:""`
	T52 ***TestBrick52 `brick:"random"`
}

func (t TestBrick52) BrickTypeID() string {
	return "TestBrick52"
}

func Test_RecursiveBrick(t *testing.T) {
	t.Run("TestBrick5", func(t *testing.T) {
		Register[*TestBrick5]()

		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("expected panic, but no panic")
			}
		}()
		Get[*TestBrick5]()
	})
	t.Run("TestBrick51", func(t *testing.T) {
		Register[*TestBrick51]()
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("expected panic, but no panic")
			}
			if strings.Contains(fmt.Sprintf("%v", r), "overflow") {
				t.Errorf("expected no overflow, but overflow")
			}
		}()
		Get[*TestBrick51]()

	})
	t.Run("TestBrick52", func(t *testing.T) {
		Register[TestBrick52]()
		defer func() {
			r := recover()
			// t.Logf("panic: %v", r)
			if r == nil {
				t.Errorf("expected panic, but no panic")
			}
			if strings.Contains(fmt.Sprintf("%v", r), "overflow") {
				t.Errorf("expected no overflow, but overflow")
			}
		}()
		Get[TestBrick52]()
	})

}

type TestBrick6 struct {
	T1 *TestBrick1 `brick:""`
	T3 TestBrick3  `brick:""`
}

func (t TestBrick6) BrickTypeID() string {
	return "TestBrick6"
}

func Test_NonPtrBrick(t *testing.T) {
	Register[TestBrick6]()
	Get[TestBrick6]()
}

func TestCreateEmptyPtrInstance(t *testing.T) {
	c := &customStruct{}
	tests := []struct {
		name     string
		typ      reflect.Type
		expected interface{}
	}{
		{
			name:     "int",
			typ:      reflect.TypeOf(int(0)),
			expected: new(int),
		},
		{
			name:     "*int",
			typ:      reflect.TypeOf(new(int)),
			expected: new(int),
		},
		{
			name:     "string",
			typ:      reflect.TypeOf(""),
			expected: new(string),
		},
		{
			name:     "*string",
			typ:      reflect.TypeOf(new(string)),
			expected: new(string),
		},
		{
			name:     "**int",
			typ:      reflect.TypeOf(new(*int)),
			expected: new(*int),
		},
		{
			name:     "struct",
			typ:      reflect.TypeOf(struct{}{}),
			expected: &struct{}{},
		},
		{
			name:     "*struct",
			typ:      reflect.TypeOf(&struct{}{}),
			expected: &struct{}{},
		},
		{
			name: "struct with fields",
			typ: reflect.TypeOf(struct {
				A int
				B string
			}{}),
			expected: &struct {
				A int
				B string
			}{},
		},
		{
			name: "*struct with fields",
			typ: reflect.TypeOf(&struct {
				A int
				B string
			}{}),
			expected: &struct {
				A int
				B string
			}{},
		},
		{
			name:     "***int",
			typ:      reflect.TypeOf(new(**int)), // ***int
			expected: new(**int),
		},
		{
			name:     "[]int",
			typ:      reflect.TypeOf([]int{}),
			expected: &[]int{}, // Note: returns a pointer to a slice
		},
		{
			name:     "*[]int",
			typ:      reflect.TypeOf(&[]int{}),
			expected: &[]int{},
		},
		{
			name:     "map[string]int",
			typ:      reflect.TypeOf(map[string]int{}),
			expected: &map[string]int{}, // Pointer to map
		},
		{
			name:     "*map[string]int",
			typ:      reflect.TypeOf(&map[string]int{}),
			expected: &map[string]int{},
		},
		{
			name:     "custom type",
			typ:      reflect.TypeOf(customType(0)),
			expected: new(customType),
		},
		{
			name:     "*custom type",
			typ:      reflect.TypeOf(new(customType)),
			expected: new(customType),
		},
		{
			name:     "custom struct type",
			typ:      reflect.TypeOf(customStruct{}),
			expected: &customStruct{},
		},
		{
			name:     "*custom struct type",
			typ:      reflect.TypeOf(&customStruct{}),
			expected: &customStruct{},
		},
		{
			name:     "**custom struct type",
			typ:      reflect.TypeOf((**customStruct)(nil)),
			expected: &c,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createEmptyPtrInstance(tt.typ)

			// Compare Types
			if got.Type() != reflect.TypeOf(tt.expected) {
				t.Errorf("createEmptyPtrInstance() type = %v, want %v", got.Type(), reflect.TypeOf(tt.expected))
			}

			// Check if it's a pointer
			if got.Kind() != reflect.Ptr {
				t.Errorf("createEmptyPtrInstance() kind = %v, want Ptr", got.Kind())
			}

			// For pointer to pointer types, check if the inner pointer is nil
			if tt.typ.Kind() == reflect.Ptr && tt.typ.Elem().Kind() == reflect.Ptr {
				if got.Elem().IsNil() {
					t.Errorf("Inner element should not be a nil pointer. Got Kind: %v", got.Elem().Elem().Kind())
				}
			}
			if tt.typ.Kind() == reflect.Ptr && tt.typ.Elem().Kind() == reflect.Ptr && tt.typ.Elem().Elem().Kind() == reflect.Ptr {
				if got.Elem().Elem().IsNil() {
					t.Errorf("Inner element should not be a nil pointer. Got Kind: %v", got.Elem().Elem().Elem().Kind())
				}
			}

		})
	}
}

type customType int
type customStruct struct {
	A int
	B string
}

type TestBrick7 struct {
	BrickBase[*TestBrick7]
	T1 *TestBrick1 `brick:""`
}

func (t *TestBrick7) BrickTypeID() string {
	return "TestBrick7"
}

type TestBrick71 struct {
	BrickBase[*TestBrick71]
	T1 *TestBrick1 `brick:""`
}

func (t TestBrick71) BrickTypeID() string {
	return "TestBrick71"
}

func Test_BaseBrick(t *testing.T) {
	Register[*TestBrick7]()
	b := Get[*TestBrick7]()
	if GetBrickTypeID[*TestBrick7]() != "TestBrick7" {
		t.Errorf("GetBrickTypeID[*TestBrick7]() = %v, want %v", GetBrickTypeID[*TestBrick7](), "TestBrick7")
	}
	if b.BrickLiveID() != GetBrickTypeID[*TestBrick7]() {
		t.Errorf("b.BrickLiveID = %v, want %v", b.BrickLiveID(), GetBrickTypeID[*TestBrick7]())
	}

	RegisterNewer[*TestBrick71]()
	b1 := Get[*TestBrick71]()
	if GetBrickTypeID[*TestBrick71]() != "TestBrick71" {
		t.Errorf("GetBrickTypeID[*TestBrick71]() = %v, want %v", GetBrickTypeID[*TestBrick71](), "TestBrick71")
	}
	if b1.BrickLiveID() != GetBrickTypeID[*TestBrick71]() {
		t.Errorf("b1.BrickLiveID = %v, want %v", b1.BrickLiveID(), GetBrickTypeID[*TestBrick71]())
	}
}
