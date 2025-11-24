# BrandishBot_Go Security Analysis

## Executive Summary

This security analysis evaluates BrandishBot_Go, a backend service designed for **local network communication** between Streamerbot and this application. The application processes **untrusted user input from chat platforms** (Twitch, YouTube, Discord) which **will not be sanitized by the sender**.

**Overall Risk Level**: **CRITICAL** - The combination of local network exposure and unsanitized user input creates severe security risks.

⚠️ **CRITICAL CONTEXT**: 
- User input comes directly from public chatrooms
- Usernames and message content contain arbitrary characters
- No sanitization occurs before reaching this service
- Platform validation is minimal/absent
- Service is accessible to entire local network

---

## Current Security Posture

### Deployment Context
- **Intended Use**: **Local network communication** (LAN)
- **Network Binding**: Server binds to `:8080` (all interfaces by default)
- **Input Source**: **Untrusted user input from public chatrooms**
- **Input Sanitization**: **NONE** - user input arrives unsanitized
- **Character Set**: Usernames and content can contain **ANY characters** supported by streaming platforms
- **No TLS/HTTPS**: Plain HTTP only
- **No Authentication**: All endpoints are publicly accessible to LAN
- **No Rate Limiting**: Unlimited requests per client
- **No Authorization**: No permission checks on operations
- **Platform Validation**: **MISSING** - accepts any string as platform name

### Existing Security Measures ✅
1. **SQL Injection Protection**: Using parameterized queries with `pgx`
2. **Input Validation**: Basic validation on required fields
3. **Structured Logging**: Request/response logging with correlation IDs
4. **Graceful Shutdown**: Proper server lifecycle management

---

## Identified Vulnerabilities

### 🔴 CRITICAL Vulnerabilities

#### 1. **No Platform Validation** 
**Severity**: CRITICAL  
**Location**: All handlers accepting platform parameter  
**Impact**: Attackers can inject arbitrary platform names, bypassing business logic

```go
// Currently accepts ANY string as platform
func updatePlatformID(user *domain.User, platform, platformID string) {
    switch platform {
    case "twitch":
        user.TwitchID = platformID
    case "youtube":
        user.YoutubeID = platformID
    case "discord":
        user.DiscordID = platformID
    // NO DEFAULT CASE - silently ignores invalid platforms!
    }
}
```

**Risk**:
- Malicious platform values could bypass security checks
- Database may store invalid platform associations
- Business logic assumes valid platforms only
- Could be used for injection attacks in future features

**Required Fix**:
```go
var validPlatforms = map[string]bool{
    "twitch":  true,
    "youtube": true,
    "discord": true,
}

func validatePlatform(platform string) error {
    if !validPlatforms[platform] {
        return fmt.Errorf("invalid platform: %s", platform)
    }
    return nil
}
```

---

#### 2. **Unsanitized Chat Input - SQL Injection Risk**
**Severity**: CRITICAL  
**Location**: All handlers processing username and item names  
**Impact**: While parameterized queries protect against SQL injection, untrusted input still poses risks

**Current Protection**: ✅ Using `pgx` parameterized queries
```go
// SAFE from SQL injection due to parameterization
query := `SELECT user_id FROM users WHERE username = $1`
err := db.QueryRow(ctx, query, req.Username).Scan(&userID)
```

**Remaining Risks**:
- **Database bloat**: Malicious users can create extremely long usernames
- **Data integrity**: Unicode exploits, homograph attacks
- **Business logic bypass**: Usernames like `"admin"`, `"system"`, `"bot"`
- **Logging injection**: Control characters in logs
- **Future features**: If data is ever used in non-parameterized contexts

**Example Attack Vectors** (from chat):
```
/command username:$(whoami)
/command username:'; DROP TABLE users; --
/command username:AAAA[... 1MB ...]AAAA
/command username:admin\x00real_user
/command username:аdmin (Cyrillic 'a')
```

**Required Mitigations**:
```go
// 1. Maximum length enforcement
const MaxUsernameLength = 100
if len(username) > MaxUsernameLength {
    return errors.New("username too long")
}

// 2. Character validation (allow platform-specific chars)
func validateUsername(username string) error {
    if username == "" || len(username) > MaxUsernameLength {
        return errors.New("invalid username length")
    }
    
    // Remove null bytes and control characters
    if strings.ContainsAny(username, "\x00\n\r\t") {
        return errors.New("username contains invalid characters")
    }
    
    return nil
}

// 3. Normalize Unicode (prevent homograph attacks)
import "golang.org/x/text/unicode/norm"
username = norm.NFC.String(username)
```

---

#### 3. **No Authentication or Authorization**
**Severity**: Critical  
**Location**: All API endpoints  
**Impact**: Any client that can reach the server can impersonate any user

```go
// Example: Anyone can add items to any user's inventory
POST /user/item/add
{
  "username": "victim_user",
  "item_name": "expensive_item",
  "quantity": 999999
}
```

**Risk**: 
- Unauthorized inventory manipulation
- User impersonation
- Stat manipulation
- Account linking abuse

**Mitigation Required**:
- Add API key authentication (minimum)
- Implement request signing for trusted clients
- Add role-based access control (RBAC) for admin operations

---

#### 4. **Server Binds to All Interfaces on Local Network**
**Severity**: CRITICAL  
**Location**: `internal/server/server.go:51`

```go
Addr: fmt.Sprintf(":%d", port),  // Binds to 0.0.0.0:8080
```

**Risk**: Server is accessible from **entire local network**, not just the host machine.

**Attack Scenarios**:
1. Any device on LAN can send requests
2. Compromised IoT devices on network
3. Guests on WiFi network
4. Malware on other computers
5. ARP spoofing/MITM on LAN

**Mitigation Required**:
- If only same-host communication needed: bind to `127.0.0.1:8080`
- If local network access needed: implement authentication (see below)
- Consider binding to specific network interface only

---

#### 3. **No TLS/HTTPS**
**Severity**: Critical (for network communication)  
**Impact**: All data transmitted in plain text

**Risk**:
- Credentials visible on network
- Session hijacking possible
- Man-in-the-middle attacks
- Data tampering

**Mitigation Required**:
- Implement TLS for any non-localhost deployment
- Use mTLS (mutual TLS) for client authentication
- Acceptable for localhost-only as encrypted loopback is generally secure

---

### 🟠 HIGH Vulnerabilities

#### 4. **No Rate Limiting**
**Severity**: High  
**Location**: No rate limiting middleware

**Risk**:
- Denial of Service (DoS) attacks
- Resource exhaustion
- Database connection pool depletion
- API abuse

**Example Attack**:
```bash
while true; do
  curl -X POST http://localhost:8080/user/item/add \
    -d '{"username":"victim","item_name":"item","quantity":1}'
done
```

**Mitigation Required**:
- Implement per-IP rate limiting
- Add per-endpoint rate limits
- Implement request queuing
- Add circuit breakers for database operations

---

#### 5. **Insufficient Input Validation**
**Severity**: High  
**Location**: Multiple handlers

**Examples**:
- No maximum length validation on usernames
- No validation on platform names (accepts any string)
- No validation on item quantities (could be negative in some cases)
- No sanitization of user-controlled strings

```go
// handlers accept arbitrary usernames with no length limits
if req.Username == "" { // Only checks for empty, not length or format
    http.Error(w, "Missing required fields", http.StatusBadRequest)
    return
}
```

**Risk**:
- Database bloat from excessively long strings
- Unexpected behavior from special characters
- Potential for injection if data is used in other contexts

**Mitigation Required**:
- Add maximum length validation (e.g., username ≤ 100 chars)
- Validate platform names against whitelist (`twitch`, `youtube`, `discord`)
- Add regex validation for expected formats
- Sanitize inputs before logging

---

#### 6. **Verbose Error Messages**
**Severity**: High  
**Location**: Multiple handlers

```go
http.Error(w, err.Error(), http.StatusInternalServerError)
```

**Risk**:
- Exposes internal implementation details
- May leak database structure or query details
- Aids attackers in reconnaissance

**Mitigation Required**:
- Return generic error messages to clients
- Log detailed errors server-side only
- Implement error codes for debugging without exposure

---

### 🟡 MEDIUM Vulnerabilities

#### 7. **No CORS Configuration**
**Severity**: Medium  
**Location**: No CORS middleware

**Risk**:
- If accessed via browser, vulnerable to CSRF
- Unauthorized cross-origin requests possible

**Current Mitigation**: Localhost-only design limits exposure  
**Recommendation**: Add CORS middleware if web frontend is planned

---

#### 8. **Database Credentials in Environment**
**Severity**: Medium  
**Location**: `.env` file, `internal/config/config.go`

**Current State**:
```env
DB_PASSWORD=pass  # Plain text in .env
```

**Risk**:
- Credentials accessible to anyone with file system access
- Credentials may be committed to version control
- No rotation mechanism

**Mitigation Required**:
- Use environment variables instead of `.env` for production
- Implement secret management (e.g., HashiCorp Vault)
- Add credential rotation procedures
- Ensure `.env` is in `.gitignore` (already done)

---

#### 9. **No Request Size Limits**
**Severity**: Medium  
**Location**: JSON decoding in handlers

**Risk**:
- Memory exhaustion from large payloads
- DoS via oversized requests

**Mitigation Required**:
```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
```

---

#### 10. **Logging Sensitive Data**
**Severity**: Medium  
**Location**: Multiple handlers log platform IDs

```go
log.Debug("Decoded request", 
    "platform", req.Platform,
    "platform_id", req.PlatformID,  // May be sensitive
    "username", req.Username)
```

**Risk**:
- Platform IDs may be considered PII
- Logs may be accessible to unauthorized parties

**Mitigation Required**:
- Redact or hash sensitive fields in logs
- Implement log levels (DEBUG only in development)
- Ensure log files have proper permissions

---

### 🟢 LOW Vulnerabilities

#### 11. **No Request ID in Responses**
**Severity**: Low  
**Impact**: Difficult to correlate client issues with server logs

**Recommendation**: Return request ID in response header for debugging

---

#### 12. **No Health/Readiness Endpoints**
**Severity**: Low  
**Impact**: Difficult to monitor service health

**Recommendation**: Add `/health` and `/ready` endpoints

---

## Local Network Security Considerations

### Threat Model: Local Network + Untrusted Chat Input

**Attackers**:
1. **Malicious chat users**: Send crafted messages via Twitch/YouTube/Discord
2. **Compromised LAN devices**: IoT devices, infected computers on network
3. **Network attackers**: ARP spoofing, packet sniffing on WiFi
4. **Insider threats**: Other users on the same network

**Attack Vectors**:
1. **Chat injection**: Malicious usernames, item names from chatroom
2. **Network access**: Direct HTTP requests from LAN devices
3. **DoS attacks**: Flood server with requests
4. **Data exfiltration**: Read other users' inventory/stats
5. **Privilege escalation**: Manipulate admin accounts

### Critical Security Requirements for Local Network

#### 1. **Input Validation is MANDATORY**
Given untrusted chat input, you MUST:

```go
// Example validation middleware
func ValidateRequest(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Limit request size
        r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
        
        // Platform validation will happen in handlers
        next.ServeHTTP(w, r)
    })
}

// In each handler
func HandleAddItem(svc user.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req AddItemRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }
        
        // REQUIRED: Platform validation
        if err := validatePlatform(req.Platform); err != nil {
            http.Error(w, "Invalid platform", http.StatusBadRequest)
            return
        }
        
        // REQUIRED: Username validation
        if err := validateUsername(req.Username); err != nil {
            http.Error(w, "Invalid username", http.StatusBadRequest)
            return
        }
        
        // REQUIRED: Item name validation
        if err := validateItemName(req.ItemName); err != nil {
            http.Error(w, "Invalid item name", http.StatusBadRequest)
            return
        }
        
        // ... rest of handler
    }
}
```

#### 2. **API Authentication is MANDATORY**
With LAN exposure, authentication is critical:

```go
// Middleware for API key validation
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            providedKey := r.Header.Get("X-API-Key")
            if providedKey != apiKey {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// In server setup
mux := http.NewServeMux()
// ... register routes
loggedMux := loggingMiddleware(mux)
authedMux := AuthMiddleware(config.APIKey)(loggedMux)

return &Server{
    httpServer: &http.Server{
        Addr:    fmt.Sprintf(":%d", port),
        Handler: authedMux,
    },
    // ...
}
```

#### 3. **Rate Limiting is MANDATORY**
Prevent DoS from LAN or chat spam:

```go
import "golang.org/x/time/rate"

// Per-IP rate limiter
type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.Mutex
}

func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    limiter, exists := rl.limiters[ip]
    if !exists {
        limiter = rate.NewLimiter(10, 20) // 10 req/sec, burst 20
        rl.limiters[ip] = limiter
    }
    
    return limiter
}

func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := strings.Split(r.RemoteAddr, ":")[0]
            limiter := rl.GetLimiter(ip)
            
            if !limiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

#### 4. **Defense in Depth Strategy**
Layer multiple security controls:

1. **Network Layer**: Firewall rules limiting source IPs
2. **Transport Layer**: Consider TLS even for LAN (prevents sniffing)
3. **Application Layer**: API key + rate limiting + input validation
4. **Data Layer**: Parameterized queries (already done) + data validation

---

## Security Recommendations for Local Network Deployment

### Immediate Actions (MUST IMPLEMENT BEFORE USE)

1. **✅ Keep Server Binding (Already Correct for LAN)**
   ```go
   // Current binding is correct for local network access
   Addr: fmt.Sprintf(":%d", port)
   ```
   
2. **🔴 IMPLEMENT Platform Validation** (CRITICAL)
   ```go
   var ValidPlatforms = map[string]bool{
       "twitch":  true,
       "youtube": true,
       "discord": true,
   }
   
   func validatePlatform(platform string) error {
       if !ValidPlatforms[platform] {
           return fmt.Errorf("unsupported platform: %s", platform)
       }
       return nil
   }
   ```
   Add to ALL handlers that accept platform parameter.

3. **🔴 IMPLEMENT Input Validation** (CRITICAL)
   ```go
   func validateUsername(username string) error {
       const MaxLen = 100
       if username == "" || len(username) > MaxLen {
           return errors.New("invalid username length")
       }
       // Remove control characters
       if strings.ContainsAny(username, "\x00\n\r\t") {
           return errors.New("invalid characters in username")
       }
       return nil
   }
   
   func validateItemName(itemName string) error {
       const MaxLen = 100
       if itemName == "" || len(itemName) > MaxLen {
           return errors.New("invalid item name length")
       }
       return nil
   }
   ```

4. **🔴 IMPLEMENT API Key Authentication**
   ```go
   // Add to config
   type Config struct {
       // ... existing fields
       APIKey string
   }
   
   // Load from environment
   cfg.APIKey = getEnv("API_KEY", "")
   if cfg.APIKey == "" {
       return nil, errors.New("API_KEY must be set")
   }
   
   // Add middleware (shown above in Local Network section)
   ```

5. **🔴 IMPLEMENT Rate Limiting**
   Use the `golang.org/x/time/rate` package (shown above)

6. **🔴 IMPLEMENT Request Size Limits**
   ```go
   r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
   ```

### Short-Term Improvements

5. **Add Rate Limiting**
   - Implement token bucket or sliding window
   - Per-endpoint and per-client limits

6. **Enhance Input Validation**
   - Add regex validation for usernames
   - Validate quantity ranges
   - Sanitize item names

7. **Improve Logging Security**
   - Redact sensitive fields
   - Implement log rotation
   - Set appropriate log file permissions

### Medium-Term Enhancements

8. **Add TLS Support** (if network deployment needed)
   - Implement HTTPS
   - Consider mTLS for client authentication

9. **Implement RBAC**
   - Admin vs. user operations
   - Platform-specific permissions

10. **Add Monitoring and Alerting**
    - Failed authentication attempts
    - Unusual request patterns
    - Resource usage monitoring

### Long-Term Considerations

11. **Security Audit**
    - Third-party code review
    - Penetration testing
    - Dependency scanning

12. **Compliance**
    - GDPR considerations (if applicable)
    - Data retention policies
    - User data protection

---

## Conclusion

**For Local Network Deployment with Untrusted Chat Input**: The current security posture is **CRITICALLY INSUFFICIENT**.

### Required Before Production Use:
1. ✅ Platform validation (whitelist twitch/youtube/discord)
2. ✅ Input validation (username, item names, quantities)
3. ✅ API key authentication
4. ✅ Rate limiting
5. ✅ Request size limits
6. ✅ Error message sanitization

### Priority Implementation Order
1. 🔴 **Platform Validation** - Prevents arbitrary platform names (1-2 hours)
2. 🔴 **Input Validation** - Mitigates chat injection attacks (2-4 hours)
3. 🔴 **API Key Auth** - Prevents unauthorized LAN access (2-3 hours)
4. 🟠 **Rate Limiting** - Prevents DoS attacks (3-4 hours)
5. 🟠 **Request Size Limits** - Prevents memory exhaustion (30 minutes)
6. 🟠 **Error Sanitization** - Prevents information disclosure (1-2 hours)

### Risk Assessment
- **Current State**: HIGH RISK - vulnerable to chat injection, unauthorized access, DoS
- **After Priority Fixes**: MEDIUM RISK - acceptable for trusted local network
- **With All Mitigations**: LOW RISK - production-ready for local network deployment

### Estimated Implementation Time
- **Minimum Viable Security**: 6-8 hours (items 1-3)
- **Production Ready**: 12-16 hours (all items)

**DO NOT DEPLOY** to production or expose to users without implementing at minimum items 1-3.
