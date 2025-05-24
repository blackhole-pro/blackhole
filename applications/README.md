# Applications Directory

This directory contains **example applications** built on top of the Blackhole Framework.

## Important: Orchestration is Application Code!

The "orchestration" or "service" layer that combines plugins is **NOT part of the framework**. It's application-specific code that each developer creates for their own needs.

```
┌─────────────────────────────────────────┐
│          YOUR APPLICATION               │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │   Your Orchestration/Services    │   │ ← YOU write this
│  │   (Application-specific logic)   │   │
│  └────────────┬────────────────────┘   │
│               │                         │
├───────────────┼─────────────────────────┤
│               ▼                         │
│        BLACKHOLE FRAMEWORK              │ ← Framework provides
│   (Plugins, Mesh, Runtime, etc.)       │   these building blocks
└─────────────────────────────────────────┘
```

## What the Framework Provides

The Blackhole Framework provides:
- **Plugins**: Reusable components (node, storage, identity, etc.)
- **Mesh Network**: Communication infrastructure
- **Plugin Management**: Loading, lifecycle, hot-swapping
- **Runtime**: Process orchestration and supervision

## What YOU Create

As an application developer, you create:
- **Orchestrators/Services**: Your business logic that combines plugins
- **Application-specific workflows**: How plugins work together for YOUR use case
- **Custom plugins**: If the existing plugins don't meet your needs

## Example Applications

### 1. Content Sharing Application
```
applications/content-sharing/
├── internal/
│   └── orchestration/          ← Application-specific orchestration
│       ├── content_distributor.go
│       └── sharing_service.go
├── cmd/
│   └── content-share/
│       └── main.go
└── README.md
```

### 2. Social Network Application
```
applications/social-network/
├── internal/
│   └── services/              ← Different name, same concept
│       ├── post_service.go
│       ├── friend_service.go
│       └── feed_service.go
├── cmd/
│   └── social/
│       └── main.go
└── README.md
```

## Key Points

1. **Orchestration is NOT in core/**: It's application code, not framework code
2. **Each app is different**: Your orchestration will be unique to your needs
3. **Full flexibility**: Combine plugins however makes sense for YOUR application
4. **No forced patterns**: The framework doesn't dictate how you structure your app

## Getting Started

To build your own application:

1. Create a new directory under `applications/`
2. Import the plugins you need from `core/pkg/plugins/`
3. Write your own orchestration logic
4. Build and run your application

Remember: The framework is just the foundation. Your application logic is yours to design!