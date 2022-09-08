package entities

import (
	"testing"

	"github.com/tj/assert"
)

func Test_checkTeamNameFormatValid(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Valid one character",
			args: args{
				name: "a",
			},
			want: true,
		},
		{
			name: "Valid one word",
			args: args{
				name: "test",
			},
			want: true,
		},
		{
			name: "Valid one dash",
			args: args{
				name: "test-1",
			},
			want: true,
		},
		{
			name: "Valid two dashes",
			args: args{
				name: "test-1-2",
			},
			want: true,
		},
		{
			name: "Valid whole alphabet",
			args: args{
				name: "abcdefghijklmnopqrstuvwxyz0123456789",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkTeamNameFormat(tt.args.name)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_checkTeamNameFormatInvalid(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Empty name",
			args: args{
				name: "",
			},
			want: false,
		},
		{
			name: "Dash in the beginning",
			args: args{
				name: "-test",
			},
			want: false,
		},
		{
			name: "Dash in the end",
			args: args{
				name: "test-",
			},
			want: false,
		},
		{
			name: "Two dashes in a row",
			args: args{
				name: "test--1",
			},
			want: false,
		},
		{
			name: "Invalid character",
			args: args{
				name: "test!",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkTeamNameFormat(tt.args.name)
			assert.Equal(t, tt.want, got)
		})
	}
}
