package manifest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildDirectories(t *testing.T) {
	wixFile := &WixManifest{}
	wixFile.Directories = []Directory{
		{
			Name: "testdata",
		},
		{
			Name: "testdata2",
		},
	}

	err := wixFile.buildDirectoriesRecursive()
	require.NoError(t, err)
	require.Equal(t, "testdata", wixFile.Directories[0].Name)

	expect := &WixManifest{}
	expect.Directories = []Directory{
		{
			Name: "testdata",
			Directories: []Directory{
				{
					Name: "path_a",
					Files: []File{
						{
							Path: "testdata/path_a/a",
						},
						{
							Path: "testdata/path_a/a2",
						},
					},
				},
				{
					Name: "path_b",
					Files: []File{
						{
							Path: "testdata/path_b/b",
						},
					},
				},
				{
					Name: "path_c",
					Directories: []Directory{
						{
							Name: "path_c_sub",
							Files: []File{
								{
									Path: "testdata/path_c/path_c_sub/csub",
								},
							},
						},
					},
				},
			},
			Files: []File{
				{
					Path: "testdata/testdata",
				},
			},
		},
		{
			Name: "testdata2",
			Directories: []Directory{
				{
					Name: "path_a",
					Files: []File{
						{
							Path: "testdata2/path_a/a",
						},
						{
							Path: "testdata2/path_a/a2",
						},
					},
				},
				{
					Name: "path_b",
					Files: []File{
						{
							Path: "testdata2/path_b/b",
						},
					},
				},
				{
					Name: "path_c",
					Directories: []Directory{
						{
							Name: "path_c_sub",
							Files: []File{
								{
									Path: "testdata2/path_c/path_c_sub/csub",
								},
							},
						},
					},
				},
			},
			Files: []File{
				{
					Path: "testdata2/testdata",
				},
			},
		},
	}

	require.Equal(t, expect, wixFile)

}

func TestEmptyDir(t *testing.T) {
	wixFile := &WixManifest{}
	d := Directory{Name: "fakedir"}
	wixFile.Directories = append(wixFile.Directories, d)

	err := wixFile.buildDirectoriesRecursive()
	require.Error(t, err)
}
