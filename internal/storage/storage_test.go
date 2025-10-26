package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	tests := []struct {
		objType  Storager
		name     string
		filePath string
		dsn      string
	}{
		{
			name:     "memory",
			filePath: "",
			dsn:      "",
			objType:  &storage{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, err := NewStorage(t.Context(), test.filePath, test.dsn)
			require.NoError(t, err)
			assert.IsType(t, test.objType, storage)
		})
	}
}

func TestGet(t *testing.T) {
	type expect struct {
		val string
		ok  bool
	}
	tests := []struct {
		name    string
		key     string
		prepare [][3]string
		expect  expect
	}{
		{
			name: "key exists",
			key:  "short",
			prepare: [][3]string{
				{"short", "long_url", "user1"},
			},
			expect: expect{
				val: "long_url",
				ok:  true,
			},
		},
		{
			name: "key not exists",
			key:  "short2",
			prepare: [][3]string{
				{"short", "long_url", "user1"},
			},
			expect: expect{
				val: "",
				ok:  false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, err := NewStorageMemory()
			require.NoError(t, err)
			if len(test.prepare) != 0 {
				for _, k := range test.prepare {
					err := storage.Set(t.Context(), k[0], k[1], k[2])
					require.NoError(t, err)
				}
			}

			val, ok, err := storage.Get(t.Context(), test.key)
			require.NoError(t, err)
			assert.Equal(t, test.expect.val, val)
			assert.Equal(t, test.expect.ok, ok)
		})
	}
}

func TestGetUserURLS(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		prepare [][3]string
		expect  []Item
	}{
		{
			name:   "url exists for user1",
			userID: "user1",
			prepare: [][3]string{
				{"short", "long_url", "user1"},
				{"short2", "long_url2", "user1"},
			},
			expect: []Item{
				{
					ShortURL:    "short",
					OriginalURL: "long_url",
				},
				{
					ShortURL:    "short2",
					OriginalURL: "long_url2",
				},
			},
		},
		{
			name:   "no urls for user2",
			userID: "user2",
			prepare: [][3]string{
				{"short", "long_url", "user1"},
				{"short2", "long_url2", "user1"},
			},
			expect: []Item{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, err := NewStorageMemory()
			require.NoError(t, err)
			if len(test.prepare) != 0 {
				for _, k := range test.prepare {
					err := storage.Set(t.Context(), k[0], k[1], k[2])
					require.NoError(t, err)
				}
			}

			items, err := storage.GetUserURLS(t.Context(), test.userID)
			require.NoError(t, err)
			assert.ElementsMatch(t, test.expect, items)
		})
	}
}

func TestGetShort(t *testing.T) {
	type expect struct {
		short string
		ok    bool
	}
	tests := []struct {
		name    string
		origURL string
		prepare [][3]string
		expect  expect
	}{
		{
			name:    "url exists",
			origURL: "long_url2",
			prepare: [][3]string{
				{"short", "long_url", "user1"},
				{"short2", "long_url2", "user1"},
			},
			expect: expect{
				short: "short2",
				ok:    true,
			},
		},
		{
			name:    "url not exists",
			origURL: "long_url3",
			prepare: [][3]string{
				{"short", "long_url", "user1"},
				{"short2", "long_url2", "user1"},
			},
			expect: expect{
				short: "",
				ok:    false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, err := NewStorageMemory()
			require.NoError(t, err)
			if len(test.prepare) != 0 {
				for _, k := range test.prepare {
					err := storage.Set(t.Context(), k[0], k[1], k[2])
					require.NoError(t, err)
				}
			}

			short, ok, err := storage.GetShort(t.Context(), test.origURL)
			require.NoError(t, err)
			assert.Equal(t, test.expect.short, short)
			assert.Equal(t, test.expect.ok, ok)
		})
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		expect  error
		name    string
		setArgs [3]string
		prepare [][3]string
	}{
		{
			name: "set without error",
			setArgs: [3]string{
				"short2", "orig2", "user1",
			},
			prepare: [][3]string{
				{"short", "long_url", "user1"},
			},
			expect: nil,
		},
		{
			name: "set without error",
			setArgs: [3]string{
				"short2", "orig2", "user2",
			},
			prepare: [][3]string{
				{"short", "long_url", "user1"},
			},
			expect: nil,
		},
		{
			name: "set with error",
			setArgs: [3]string{
				"short", "long_url", "user1",
			},
			prepare: [][3]string{
				{"short", "long_url", "user1"},
			},
			expect: ErrConflict,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, err := NewStorageMemory()
			require.NoError(t, err)
			if len(test.prepare) != 0 {
				for _, k := range test.prepare {
					err := storage.Set(t.Context(), k[0], k[1], k[2])
					require.NoError(t, err)
				}
			}

			err = storage.Set(t.Context(), test.setArgs[0], test.setArgs[1], test.setArgs[2])
			assert.Equal(t, test.expect, err)
		})
	}
}
