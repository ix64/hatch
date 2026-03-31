package projectmeta

import "encoding/json"

// JSONSchema returns the JSON Schema for hatch.toml and hatch.local.toml metadata files.
func JSONSchema() ([]byte, error) {
	doc := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://github.com/ix64/hatch/raw/main/hatch.schema.json",
		"title":                "Hatch Project Metadata",
		"description":          "Schema for hatch.toml and hatch.local.toml files consumed by the hatch CLI.",
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"module_path":        stringSchema("Go module path for the application, for example example.com/acme/demo.", "example.com/acme/demo"),
			"app_name":           stringSchema("Human-friendly application name shown in generated docs and CLI output.", "Demo Service"),
			"binary_name":        stringSchema("Binary name used for build output and generated scripts.", "demo"),
			"go_version":         stringSchemaWithDefault("Go toolchain version used by generated projects.", DefaultGo),
			"hatch_module_path":  stringSchemaWithDefault("Module path for the hatch framework dependency.", DefaultHatchMod),
			"hatch_version":      stringSchemaWithDefault("Version selector for the hatch framework dependency in generated go.mod files.", DefaultHatchVer),
			"hatch_replace_path": stringSchema("Optional local replace path used during hatch development.", "/path/to/hatch"),
			"proto_enabled": map[string]any{
				"type":        "boolean",
				"description": "Legacy top-level proto enable flag. Prefer [proto].enabled in new files.",
				"deprecated":  true,
			},
			"paths": map[string]any{
				"$ref": "#/$defs/paths",
			},
			"ent": map[string]any{
				"$ref": "#/$defs/ent",
			},
			"run": map[string]any{
				"$ref": "#/$defs/run",
			},
			"proto": map[string]any{
				"$ref": "#/$defs/proto",
			},
		},
		"$defs": map[string]any{
			"paths": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"description":          "Path configuration for generated project files and directories.",
				"properties": map[string]any{
					"main_package":   stringSchemaWithDefault("Main server package used by build and dev commands.", "./cmd/server"),
					"build_output":   stringSchema("Build output path for the compiled application binary.", "build/demo"),
					"config_file":    stringSchemaWithDefault("Runtime config file path.", "config.toml"),
					"config_example": stringSchemaWithDefault("Example config file path committed to the repository.", "config.toml.example"),
					"atlas_file":     stringSchemaWithDefault("Atlas environment configuration file.", "atlas.hcl"),
					"dev_compose":    stringSchemaWithDefault("Docker Compose file used by hatch env commands.", "dev/compose.yaml"),
					"schema_dir":     stringSchemaWithDefault("Directory containing Ent schema source files.", "ddl/schema"),
					"composite_dir":  stringSchemaWithDefault("Directory containing composite schema dumps for Atlas diffs.", "ddl/composite"),
					"migrations_dir": stringSchemaWithDefault("Directory containing Atlas migration files.", "ddl/migrations"),
					"ent_dir":        stringSchemaWithDefault("Directory where generated Ent code is written.", "ddl/ent"),
					"buf_gen_file": map[string]any{
						"type":        "string",
						"description": "Legacy path field retained for backward compatibility.",
						"deprecated":  true,
					},
					"rpc_dir": map[string]any{
						"type":        "string",
						"description": "Legacy proto output directory field. Prefer [proto].out_dir in new files.",
						"deprecated":  true,
					},
				},
			},
			"ent": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"description":          "Ent feature flags applied during code generation.",
				"properties": map[string]any{
					"features": map[string]any{
						"type":        "array",
						"description": "Ent feature flags enabled for the project.",
						"items": map[string]any{
							"type": "string",
						},
						"default": DefaultEntFeatures,
					},
				},
			},
			"run": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"description":          "Default command arguments used by hatch start and hatch dev.",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "array",
						"description": "Command and arguments passed to the generated server binary.",
						"items": map[string]any{
							"type": "string",
						},
						"default": []string{"serve"},
					},
				},
			},
			"proto": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"description":          "Proto source configuration used by hatch gen rpc.",
				"properties": map[string]any{
					"enabled":            boolSchemaWithDefault("Whether RPC generation is enabled for the project.", true),
					"source":             enumStringSchema("Where proto sources come from.", "local", []string{"local", "git"}),
					"dir":                stringSchemaWithDefault("Directory containing local proto source files.", "proto"),
					"out_dir":            stringSchemaWithDefault("Directory where generated RPC code is written.", "rpc"),
					"git_repo":           stringSchema("Git repository URL used when proto.source is git.", "https://github.com/acme/contracts.git"),
					"git_branch":         stringSchemaWithDefault("Git branch used when proto.source is git.", "main"),
					"local_override_dir": stringSchema("Optional local directory that overrides a git-based proto source on the current machine.", "../contracts"),
				},
				"allOf": []any{
					map[string]any{
						"if": map[string]any{
							"properties": map[string]any{
								"source": map[string]any{
									"const": "git",
								},
							},
							"required": []string{"source"},
						},
						"then": map[string]any{
							"required": []string{"git_repo"},
						},
					},
				},
			},
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}

func stringSchema(description string, example string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
		"examples":    []string{example},
	}
}

func stringSchemaWithDefault(description string, defaultValue string) map[string]any {
	schema := stringSchema(description, defaultValue)
	schema["default"] = defaultValue
	return schema
}

func boolSchemaWithDefault(description string, defaultValue bool) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
		"default":     defaultValue,
	}
}

func enumStringSchema(description string, defaultValue string, values []string) map[string]any {
	schema := stringSchemaWithDefault(description, defaultValue)
	schema["enum"] = values
	return schema
}
