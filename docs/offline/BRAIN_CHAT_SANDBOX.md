# Brain Chat Sandbox

Brain Chat Sandbox is a hidden/advanced local Gemma chat surface inside the GUI.

It is for explanation, report summarization, and safe next-step suggestions. It is not an executor.

## Rules

- Cannot run commands.
- Cannot read files unless the user pasted the text.
- Cannot change system settings.
- Cannot collect secrets.
- Cannot invoke recipes.
- Cannot claim it repaired the machine.

If the user asks for repair, the sandbox should direct them to Guided Rescue or Expert Console.

## CLI

```bash
./bin/agentlink brain chat --prompt "Explain this report in one sentence." --json
```

The command uses local Gemma through `llama-cli` when Brain assets are present. If the model/runtime is missing, it fails cleanly.
