# Repro Lootbox1 Usage

$baseUrl = "http://localhost:8080"
$username = "debug_user"
$platform = "twitch"
$platformId = "debug_123"

# 1. Register User (via HandleMessage)
Write-Host "Registering user..."
$body = @{
    username = $username
    platform = $platform
    platform_id = $platformId
} | ConvertTo-Json
Invoke-RestMethod -Uri "$baseUrl/message/handle" -Method Post -Body $body -ContentType "application/json"

# 2. Give Lootbox1
Write-Host "Giving lootbox1..."
$body = @{
    ownerUsername = "admin" # Assuming admin exists or we can just give to self if endpoint allows
    receiverUsername = $username
    platform = $platform
    itemName = "lootbox1"
    quantity = 1
} | ConvertTo-Json
# Wait, /user/item/give requires owner to have it? 
# Let's check /user/item/add (admin endpoint?)
# server.go: mux.HandleFunc("/user/item/add", handler.HandleAddItem(userService))
# Let's use AddItem to inject it.

$body = @{
    username = $username
    platform = $platform
    itemName = "lootbox1"
    quantity = 1
} | ConvertTo-Json
Invoke-RestMethod -Uri "$baseUrl/user/item/add" -Method Post -Body $body -ContentType "application/json"

# 3. Use Lootbox1
Write-Host "Using lootbox1..."
$body = @{
    username = $username
    platform = $platform
    itemName = "lootbox1"
    quantity = 1
} | ConvertTo-Json
try {
    $response = Invoke-RestMethod -Uri "$baseUrl/user/item/use" -Method Post -Body $body -ContentType "application/json"
    Write-Host "Response: $($response.message)"
} catch {
    Write-Host "Error using item: $_"
    exit 1
}

# 4. Check Inventory
Write-Host "Checking inventory..."
$inventory = Invoke-RestMethod -Uri "$baseUrl/user/inventory?username=$username" -Method Get
$lootbox0 = $inventory | Where-Object { $_.name -eq "lootbox0" }

if ($lootbox0) {
    Write-Host "SUCCESS: Found lootbox0 in inventory."
} else {
    Write-Host "FAILURE: lootbox0 not found."
}
