package ansiblejinja

import (
	"strings"
	"testing"
)

func TestForLoop(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "simple for loop over slice",
			template: "{% for item in items %}{{ item }},{% endfor %}",
			context: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			want:    "a,b,c,",
			wantErr: false,
		},
		{
			name:     "for loop over empty list",
			template: "Items: {% for item in items %}{{ item }},{% endfor %}",
			context: map[string]interface{}{
				"items": []interface{}{},
			},
			want:    "Items: ",
			wantErr: false,
		},
		{
			name:     "for loop with loop variable",
			template: "{% for i in items %}{{ loop.index }}:{{ i }},{% endfor %}",
			context: map[string]interface{}{
				"items": []interface{}{10, 20, 30},
			},
			want:    "1:10,2:20,3:30,",
			wantErr: false,
		},
		{
			name:     "for loop with conditional",
			template: "{% for i in items %}{% if loop.index > 1 %},{% endif %}{{ i }}{% endfor %}",
			context: map[string]interface{}{
				"items": []interface{}{10, 20, 30},
			},
			want:    "10,20,30",
			wantErr: false,
		},
		{
			name:     "nested for loops",
			template: "{% for i in outer %}[{% for j in inner %}{{ i }}-{{ j }}{% if not loop.last %},{% endif %}{% endfor %}]{% endfor %}",
			context: map[string]interface{}{
				"outer": []interface{}{"a", "b"},
				"inner": []interface{}{1, 2, 3},
			},
			want:    "[a-1,a-2,a-3][b-1,b-2,b-3]",
			wantErr: false,
		},
		{
			name:     "for loop over string",
			template: "{% for char in text %}{{ char }}{% endfor %}",
			context: map[string]interface{}{
				"text": "abc",
			},
			want:    "abc",
			wantErr: false,
		},
		{
			name:     "for loop over map values",
			template: "{% for value in user %}{{ value }}{% if not loop.last %},{% endif %}{% endfor %}",
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  30,
				},
			},
			want:    "Alice,30", // Order may vary, so this test is simplified
			wantErr: false,
		},
		{
			name:     "for loop with object attributes",
			template: "{% for user in users %}{{ user.name }} ({{ user.age }}){% if not loop.last %}, {% endif %}{% endfor %}",
			context: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"name": "Alice", "age": 30},
					map[string]interface{}{"name": "Bob", "age": 25},
				},
			},
			want:    "Alice (30), Bob (25)",
			wantErr: false,
		},
		{
			name:     "for loop over nil",
			template: "{% for item in nil_var %}{{ item }}{% endfor %}",
			context: map[string]interface{}{
				"nil_var": nil,
			},
			want:    "",
			wantErr: false,
		},
		{
			name:     "invalid for loop syntax",
			template: "{% for item items %}{{ item }}{% endfor %}",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "unclosed for loop",
			template: "{% for item in items %}{{ item }}",
			context: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			want:    "",
			wantErr: true,
		},
		{
			name:     "non-iterable in for loop",
			template: "{% for item in number %}{{ item }}{% endfor %}",
			context: map[string]interface{}{
				"number": 42,
			},
			want:    "",
			wantErr: true,
		},
		{
			name:     "simple for loop test",
			template: "{% for i in [1, 2, 3] %}{{ i }}{% endfor %}",
			context:  map[string]interface{}{},
			want:     "123",
			wantErr:  false,
		},
		{
			name:     "basic loop index test",
			template: "{% for i in [10, 20, 30] %}i={{ i }} index={{ loop.index }}|{% endfor %}",
			context:  map[string]interface{}{},
			want:     "i=10 index=1|i=20 index=2|i=30 index=3|",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TemplateString(tt.template, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplateString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// For map iteration test, the order of values is non-deterministic
				// So just check that all expected values are present
				if strings.Contains(tt.name, "map values") {
					for _, val := range []string{"Alice", "30"} {
						if !strings.Contains(got, val) {
							t.Errorf("TemplateString() got = %v, expected to contain %v", got, val)
						}
					}
				} else if got != tt.want {
					// Debug specific test cases
					if tt.name == "for_loop_with_loop_variable" {
						// Get the loop values directly
						testCtx := make(map[string]interface{})
						for k, v := range tt.context {
							testCtx[k] = v
						}
						loopVar := map[string]interface{}{
							"index": 1,
							"last":  false,
						}
						testCtx["loop"] = loopVar

						// Try to evaluate loop.index directly
						indexVal, err := EvaluateExpression("loop.index", testCtx)
						lastVal, err2 := EvaluateExpression("loop.last", testCtx)
						t.Logf("Debug - loop.index=%v (err=%v), loop.last=%v (err=%v)",
							indexVal, err, lastVal, err2)
					}
					t.Errorf("TemplateString() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestForLoopComplex(t *testing.T) {
	template := `Users:
{% for user in users %}
Username: {{ user.username }}
Email: {{ user.email }}
Roles: {% for role in user.permissions.roles %}{{ role }}{% if not loop.last %}, {% endif %}{% endfor %}
{% endfor %}`

	context := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"username": "admin",
				"email":    "admin@example.com",
				"permissions": map[string]interface{}{
					"roles": []interface{}{"admin", "user", "editor"},
				},
			},
			map[string]interface{}{
				"username": "guest",
				"email":    "guest@example.com",
				"permissions": map[string]interface{}{
					"roles": []interface{}{"guest"},
				},
			},
		},
	}

	expected := `Users:

Username: admin
Email: admin@example.com
Roles: admin, user, editor

Username: guest
Email: guest@example.com
Roles: guest
`

	result, err := TemplateString(template, context)
	if err != nil {
		t.Errorf("TemplateString() failed with error: %v", err)
		return
	}

	if result != expected {
		t.Errorf("TemplateString() complex for loop failed:\nGot:\n%s\nWant:\n%s", result, expected)
	}
}
