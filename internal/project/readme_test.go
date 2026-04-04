package project

import "testing"

func TestBuildReadme(t *testing.T) {
	tests := []struct {
		name  string
		title string
		tags  []string
		want  string
	}{
		{
			name:  "title and tags",
			title: "My Project",
			tags:  []string{"cli", "golang"},
			want:  "---\ntitle: My Project\ntags: [cli, golang]\n---\n\n# My Project\n",
		},
		{
			name:  "title only",
			title: "My Project",
			tags:  nil,
			want:  "---\ntitle: My Project\n---\n\n# My Project\n",
		},
		{
			name:  "single tag",
			title: "Foo",
			tags:  []string{"bar"},
			want:  "---\ntitle: Foo\ntags: [bar]\n---\n\n# Foo\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildReadme(tt.title, tt.tags)
			if got != tt.want {
				t.Errorf("BuildReadme:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
