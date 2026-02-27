# C4 Model (Structurizr DSL)

This directory contains the C4 architecture model for the **Event Booking System** as [Structurizr DSL](https://docs.structurizr.com/dsl). It is the single source of truth for:

- **System Context** – Event Attendee, Event Organizer, Event Booking System
- **Containers** – Attendee Mobile App, Organizer Check-in App, Organizer Web Portal, Backend API, Database
- **Components** – Backend API internals (controllers, services, repositories, scheduler, WebSocket)

The rendered diagrams are also available as Mermaid in [../C4 Diagrams/](../C4%20Diagrams/).

## Files

| File | Purpose |
|------|--------|
| `workspace.dsl` | Full workspace: model (elements + relationships) and views (system context, container, component diagrams) |

## Viewing and editing

### Structurizr Lite (recommended)

Run Lite with this directory mounted so it picks up `workspace.dsl` and hot-reloads on save:

```bash
# From repo root
make c4-lite
```

Or with Docker directly:

```bash
# From repo root
docker run -it --rm -p 8081:8080 \
  -v "$(pwd)/docs/c4:/usr/local/structurizr" \
  structurizr/lite
```

Then open **http://localhost:8081** in your browser.

### Structurizr Playground

Paste the contents of `workspace.dsl` into [Structurizr Playground](https://playground.structurizr.com/) to view and experiment without installing anything.

## Validating and exporting

Install the [Structurizr CLI](https://docs.structurizr.com/cli), then from the repo root:

```bash
# Validate the DSL
structurizr-cli validate -workspace docs/c4/workspace.dsl

# Export to JSON (e.g. for other tools)
structurizr-cli export -workspace docs/c4/workspace.dsl -format json -output docs/c4/

# Export diagrams to Mermaid (requires workspace JSON first)
structurizr-cli export -workspace docs/c4/workspace.dsl -format json -output docs/c4/
structurizr-cli export -workspace docs/c4/workspace.json -format mermaid -output docs/C4\ Diagrams/
```

After exporting to Mermaid, you can update the code blocks in `Context.md`, `Container.md`, and `Component-BackendAPI.md` with the generated files if you want the Markdown docs to stay in sync with the DSL.

## DSL rules (Structurizr)

- **Identifiers:** `!identifiers hierarchical` is set, so nested elements are referred to like `eventSystem.backendApi.scheduleCtrl`.
- **Braces:** Opening `{` on the same line as the keyword; closing `}` on its own line.
- **Relationships:** Between the same two elements, each relationship must have a unique description.

## References

- [Structurizr DSL docs](https://docs.structurizr.com/dsl)
- [Structurizr DSL language reference](https://docs.structurizr.com/dsl/language)
- [C4 model](https://c4model.com/)
