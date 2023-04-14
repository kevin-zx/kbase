package kfile

import (
	"testing"
)

// func TestFileExist(t *testing.T) {
//   type args struct {
//     filename string
//   }
//   tests := []struct {
//     name string
//     args args
//     want bool
//   }{
//     // TODO: Add test cases.
//   }
//   for _, tt := range tests {
//     t.Run(tt.name, func(t *testing.T) {
//       if got := FileExist(tt.args.filename); got != tt.want {
//         t.Errorf("FileExist() = %v, want %v", got, tt.want)
//       }
//     })
//   }
// }
//
// func TestMkdir(t *testing.T) {
//   type args struct {
//     dir string
//   }
//   tests := []struct {
//     name    string
//     args    args
//     wantErr bool
//   }{
//     // TODO: Add test cases.
//   }
//   for _, tt := range tests {
//     t.Run(tt.name, func(t *testing.T) {
//       if err := Mkdir(tt.args.dir); (err != nil) != tt.wantErr {
//         t.Errorf("Mkdir() error = %v, wantErr %v", err, tt.wantErr)
//       }
//     })
//   }
// }
//
// func TestGetFilesFromDir(t *testing.T) {
//   type args struct {
//     dir string
//   }
//   tests := []struct {
//     name    string
//     args    args
//     want    []string
//     wantErr bool
//   }{
//     // TODO: Add test cases.
//   }
//   for _, tt := range tests {
//     t.Run(tt.name, func(t *testing.T) {
//       got, err := GetFilesFromDir(tt.args.dir)
//       if (err != nil) != tt.wantErr {
//         t.Errorf("GetFilesFromDir() error = %v, wantErr %v", err, tt.wantErr)
//         return
//       }
//       if !reflect.DeepEqual(got, tt.want) {
//         t.Errorf("GetFilesFromDir() = %v, want %v", got, tt.want)
//       }
//     })
//   }
// }

func TestGetPureFilename(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple",
			args: args{
				filename: "test.txt",
			},
			want: "test",
		},
		{
			name: "with path",
			args: args{
				filename: "/home/kevin/test.txt",
			},
			want: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPureFilename(tt.args.filename); got != tt.want {
				t.Errorf("GetPureFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func TestGetFileExt(t *testing.T) {
//   type args struct {
//     filename string
//   }
//   tests := []struct {
//     name string
//     args args
//     want string
//   }{
//     // TODO: Add test cases.
//   }
//   for _, tt := range tests {
//     t.Run(tt.name, func(t *testing.T) {
//       if got := GetFileExt(tt.args.filename); got != tt.want {
//         t.Errorf("GetFileExt() = %v, want %v", got, tt.want)
//       }
//     })
//   }
// }
