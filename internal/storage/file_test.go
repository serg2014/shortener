package storage

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_File_newStorageIO(t *testing.T) {
	tests := []struct {
		name     string
		fileData io.ReadWriter
	}{
		{
			name: "one",
			fileData: bytes.NewBufferString(
				`{"short_url":"short", "original_url":"orig","user_id":"user1"}
				{"short_url":"short2", "original_url":"orig2","user_id":"user2"}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, err := newStorageIO(test.fileData)
			require.NoError(t, err)
			val, ok, err := storage.Get(t.Context(), "short")
			require.NoError(t, err)
			assert.Equal(t, "orig", val)
			assert.Equal(t, true, ok)

			val, ok, err = storage.Get(t.Context(), "short2")
			require.NoError(t, err)
			assert.Equal(t, "orig2", val)
			assert.Equal(t, true, ok)

			val, ok, err = storage.Get(t.Context(), "short3")
			require.NoError(t, err)
			assert.Equal(t, "", val)
			assert.Equal(t, false, ok)
		})
	}
}

func Test_File_Set(t *testing.T) {
	tests := []struct {
		name     string
		fileData io.ReadWriter
		setArgs  [][3]string
		expect   error
		writen   string
	}{
		{
			name: "set without error",
			fileData: bytes.NewBufferString(
				`{"short_url":"short", "original_url":"orig","user_id":"user1"}
				{"short_url":"short2", "original_url":"orig2","user_id":"user2"}`),
			setArgs: [][3]string{
				{"short3", "orig3", "user3"},
				{"short4", "orig4", "user4"},
				{"short5", "orig5", "user5"},
			},
			expect: nil,
			writen: `{"short_url":"short3","original_url":"orig3","user_id":"user3"}
{"short_url":"short4","original_url":"orig4","user_id":"user4"}
{"short_url":"short5","original_url":"orig5","user_id":"user5"}
`,
		},
		{
			name: "set with error",
			fileData: bytes.NewBufferString(
				`{"short_url":"short", "original_url":"orig","user_id":"user1"}
				{"short_url":"short2", "original_url":"orig2","user_id":"user2"}`),
			setArgs: [][3]string{
				{"short3", "orig", "user1"},
			},
			expect: ErrConflict,
			writen: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, err := newStorageIO(test.fileData)
			require.NoError(t, err)

			for _, insert := range test.setArgs {
				err = storage.Set(t.Context(), insert[0], insert[1], insert[2])
				assert.Equal(t, test.expect, err)
			}
			allRows, err := readAllDataInTest(test.fileData)
			require.NoError(t, err)
			assert.Equal(t, test.writen, allRows)
		})
	}
}

func readAllDataInTest(file io.ReadWriter) (string, error) {
	allRows := make([]byte, 0)
	p := make([]byte, 10)
	for i := true; i == true; {
		n, err := file.Read(p)
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			i = false
		}
		allRows = append(allRows, p[:n]...)
	}
	return string(allRows), nil
}

func Test_File_SetBatch(t *testing.T) {
	tests := []struct {
		name     string
		UserID   string
		fileData io.ReadWriter
		data     Short2orig
		expect   string
	}{
		{
			name:     "First",
			UserID:   "user1",
			fileData: bytes.NewBuffer(nil),
			data: Short2orig{
				"a1234567": "http://one.ru",
				"a1234568": "http://two.ru",
				"a1234569": "http://three.ru",
			},
			expect: `{"short_url":"a1234567","original_url":"http://one.ru","user_id":"user1"}
{"short_url":"a1234568","original_url":"http://two.ru","user_id":"user1"}
{"short_url":"a1234569","original_url":"http://three.ru","user_id":"user1"}
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fileStorage, err := newStorageIO(test.fileData)
			require.NoError(t, err)

			err = fileStorage.SetBatch(t.Context(), test.data, test.UserID)
			require.NoError(t, err)

			data, err := readAllDataInTest(test.fileData)
			require.NoError(t, err)
			assert.ElementsMatch(t, strings.Split(test.expect, "\n"), strings.Split(data, "\n"))
		})
	}
}
