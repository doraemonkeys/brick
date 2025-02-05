package brick

import (
	"os/exec"
	"reflect"
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
		}()
		Get[*TestBrick51]()

	})
}
