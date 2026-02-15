# C# Client Development Guidelines

This document provides guidelines for developing and extending the BrandishBot C# client (`client/csharp/`).

## Architecture

The C# client is designed as a collection of partial classes to allow for modular organization while presenting a unified API to the user (Streamer.bot).

### File Organization

- **`BrandishBotClient.cs`**: Contains the core `BrandishBotClient` class definition, initialization logic, and base HTTP methods (`GetAsync`, `PostAsync`).
- **`BrandishBotClient.Core.cs`**: Contains core functionality like `RegisterUser`, `HandleMessage`, `GetInfo`, etc.
- **`BrandishBotClient.[Feature].cs`**: Contains feature-specific methods (e.g., `BrandishBotClient.Inventory.cs`, `BrandishBotClient.Economy.cs`).
- **`Models.cs`**: Contains data models used for requests and responses.
- **`ResponseFormatter.cs`**: Contains logic for formatting API responses into user-friendly strings.

## Contributing

When adding new features or methods to the client, follow these guidelines:

### 1. Partial Classes

Always extend the `BrandishBotClient` using partial classes. Create a new file `BrandishBotClient.[Feature].cs` if adding a new feature domain.

### 2. URL Encoding

**CRITICAL requirement:** All user-supplied string parameters used in query strings MUST be URL-encoded using `System.Uri.EscapeDataString`.

The `BuildQuery` helper method in `BrandishBotClient.cs` simply joins strings and **does not** perform encoding. You must encode values *before* passing them to `BuildQuery` or when constructing query strings manually.

#### Incorrect:
```csharp
// ❌ VULNERABLE: Special characters in username will break the query
var query = BuildQuery("platform=" + platform, "username=" + username);
```

#### Correct:
```csharp
// ✅ SAFE: All parameters are properly encoded
var query = BuildQuery(
    "platform=" + System.Uri.EscapeDataString(platform),
    "username=" + System.Uri.EscapeDataString(username)
);
```

### 3. Examples

See `client/csharp/BrandishBotClient.Compost.cs` or `client/csharp/BrandishBotClient.Actions.cs` for correct usage examples where `System.Uri.EscapeDataString` is applied.

### 4. Method Naming

- Use `Async` suffix for internal asynchronous methods if they are not exposed directly to Streamer.bot (though currently most methods are exposed as `Task<T>`).
- Follow C# naming conventions (PascalCase for methods).

### 5. Error Handling

Use `HandleHttpResponse<T>` or `HandleHttpResponse` to process API responses. These helpers automatically parse error messages from the API and throw informative exceptions.

## Checklist for New Methods

- [ ] Method is async and returns `Task<T>`.
- [ ] HTTP verb matches the API endpoint (GET, POST).
- [ ] Endpoint path is correct.
- [ ] **Query parameters are URL-encoded.**
- [ ] Request body is a proper C# object (anonymous or typed).
- [ ] Response type matches the API response structure.
