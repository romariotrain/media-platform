# Create all Kafka topics for Media Platform
# PowerShell version for Windows

Write-Host "Creating Kafka topics..." -ForegroundColor Yellow
Write-Host ""

# Media Service topics
Write-Host "[Media Service] topics..." -ForegroundColor Cyan
docker exec mp_kafka kafka-topics --create --bootstrap-server localhost:9092 `
  --topic events.media `
  --partitions 1 --replication-factor 1 --if-not-exists

# Quota Service topics
Write-Host ""
Write-Host "[Quota Service] topics..." -ForegroundColor Cyan

# Commands (lowercase)
Write-Host "Creating commands.quota.reserve..."
docker exec mp_kafka kafka-topics --create --bootstrap-server localhost:9092 `
  --topic commands.quota.reserve `
  --partitions 1 --replication-factor 1 --if-not-exists

Write-Host "Creating commands.quota.release..."
docker exec mp_kafka kafka-topics --create --bootstrap-server localhost:9092 `
  --topic commands.quota.release `
  --partitions 1 --replication-factor 1 --if-not-exists

# Events (lowercase)
Write-Host "Creating events.quota.reserved..."
docker exec mp_kafka kafka-topics --create --bootstrap-server localhost:9092 `
  --topic events.quota.reserved `
  --partitions 1 --replication-factor 1 --if-not-exists

Write-Host "Creating events.quota.released..."
docker exec mp_kafka kafka-topics --create --bootstrap-server localhost:9092 `
  --topic events.quota.released `
  --partitions 1 --replication-factor 1 --if-not-exists

Write-Host "Creating events.quota.failed..."
docker exec mp_kafka kafka-topics --create --bootstrap-server localhost:9092 `
  --topic events.quota.failed `
  --partitions 1 --replication-factor 1 --if-not-exists

Write-Host ""
Write-Host "All topics created!" -ForegroundColor Green
Write-Host ""
Write-Host "List of topics:" -ForegroundColor Cyan
docker exec mp_kafka kafka-topics --list --bootstrap-server localhost:9092
