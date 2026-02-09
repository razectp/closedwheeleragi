# 🧠 Brain, Roadmap & Health - Complete Guide

## 📖 Overview

This guide documents three integrated features that transform the agent into a truly reflective and strategic system:

1. **🧠 Brain** - Transparent knowledge base for learning
2. **🗺️ Roadmap** - Long-term strategic planning
3. **🏥 Health Check** - Project state monitoring and reflection

## 🧠 Brain - Knowledge Base

### What Is It?

The Brain is a knowledge base in Markdown (`workplace/brain.md`) where the agent records:
- **Errors and Solutions** - Problems encountered and how they were resolved
- **Code Patterns** - Conventions and patterns discovered
- **Architectural Decisions** - Important technical choices
- **Insights** - General observations and discoveries

### Why Is It Important?

Unlike invisible JSON memory, the Brain allows:
- ✅ **Visibility** - You see what the agent is learning
- ✅ **Persistence** - Knowledge survives between sessions
- ✅ **Manual Editing** - You can add your own lessons
- ✅ **Natural Learning** - Structured and readable format

### How Does It Work?

#### Automatic Initialization

The `workplace/brain.md` file is created automatically when the agent starts:

```markdown
# 🧠 Agent Knowledge Base

## Errors and Solutions
<!-- Errors found and how they were resolved -->

## Code Patterns
<!-- Patterns and conventions discovered in the project -->

## Architectural Decisions
<!-- Important technical decisions made -->

## Insights
<!-- General observations and discoveries -->
```

#### Automatic Recording

The agent records automatically:

**During errors:**
```go
brain.AddError(
    "Build Failed After Dependency Update",
    "Error: undefined reference to 'NewFeature'",
    "Added missing import statement",
    []string{"build", "dependency"},
)
```

**During discoveries:**
```go
brain.AddPattern(
    "Always Use Mutex for Shared State",
    "Detected race condition in concurrent access",
    []string{"concurrency", "pattern"},
)
```

**During decisions:**
```go
brain.AddDecision(
    "Use gRPC for Microservices",
    "Migrating from REST to gRPC",
    "Better performance and type safety",
    []string{"architecture", "grpc"},
)
```

### Public API

#### Via Code

```go
// Get brain
brain := agent.GetBrain()

// Add error and solution
brain.AddError(
    "Database Connection Timeout",
    "Connection timed out after 30s",
    "Increased timeout to 60s and added retry logic",
    []string{"database", "performance"},
)

// Add pattern
brain.AddPattern(
    "Repository Pattern for Data Access",
    "All data access goes through repository layer",
    []string{"architecture", "pattern"},
)

// Add decision
brain.AddDecision(
    "Migrate to PostgreSQL",
    "Moving from SQLite to PostgreSQL",
    "Better scalability and concurrent access",
    []string{"database", "migration"},
)

// Add insight
brain.AddInsight(
    "Test Coverage Improved Performance",
    "Adding tests revealed N+1 queries",
    []string{"testing", "performance"},
)

// Read content
content, _ := brain.Read()

// Search entries
matches, _ := brain.Search("database")
```

#### Via Telegram

```
/status - View brain summary
```

### Entry Example

```markdown
### Build Failed After Dependency Update
*2026-02-09 14:30*

Error: undefined reference to 'NewFeature' in pkg/llm/client.go

**Solution:** Added missing import statement for the new package

**Tags:** `build`, `dependency`
```

---

## 🗺️ Roadmap - Strategic Planning

### What Is It?

The Roadmap is a strategic document (`workplace/roadmap.md`) that goes beyond tactical tasks in `task.md`, including:
- **Long-term Vision** - Strategic objectives
- **Milestones** - Important milestones
- **Prioritized Objectives** - High, Medium, Low priority
- **Dependencies** - Relationships between objectives
- **Status Tracking** - Planned, In Progress, Blocked, Completed

### Why Is It Important?

- 📊 **Strategic Vision** - Planning beyond day-to-day
- 🎯 **Alignment** - Everyone works toward the same goals
- 📈 **Measurable Progress** - Tracking achievements
- 🚀 **Focus** - Clear prioritization of efforts

### How Does It Work?

#### Automatic Initialization

```markdown
# 🗺️ Strategic Roadmap

## 🎯 Vision
<!-- Long-term vision -->

## 🏆 Milestones
<!-- Important milestones -->

## 📊 Strategic Objectives

### 🔴 High Priority
<!-- Critical objectives -->

### 🟡 Medium Priority
<!-- Important objectives -->

### 🟢 Low Priority
<!-- Future objectives -->

## ✅ Completed
<!-- Achievements -->

## 🚫 Blocked
<!-- Blockers and reasons -->
```

#### Adding Objectives

```go
roadmap := agent.GetRoadmap()

goal := roadmap.Goal{
    ID:          "perf-2026-q1",
    Title:       "Optimize LLM Token Usage",
    Description: "Reduce token consumption by 50%",
    Status:      "in-progress",
    Priority:    "high",
    DueDate:     &targetDate,
    Dependencies: []string{"cache-2026-q1"},
    Tags:        []string{"performance", "cost"},
    CreatedAt:   time.Now(),
}

roadmap.AddGoal(goal)
```

#### Adding Milestones

```go
milestone := roadmap.Milestone{
    Title:       "v1.0 Production Ready",
    Description: "All critical features stable",
    Goals:       []string{"perf-2026-q1", "security-2026-q1"},
    TargetDate:  &releaseDate,
}

roadmap.AddMilestone(milestone)
```

#### Updating Status

```go
// Move objective to "completed"
roadmap.UpdateGoalStatus("perf-2026-q1", "completed")

// Move to "blocked"
roadmap.UpdateGoalStatus("cache-2026-q1", "blocked")
```

### Public API

```go
// Get roadmap
roadmap := agent.GetRoadmap()

// Read full content
content, _ := roadmap.Read()

// View summary
summary, _ := roadmap.GetSummary()
// Output:
// 📊 Roadmap Status:
// - High Priority: 3 objectives
// - Medium Priority: 5 objectives
// - Low Priority: 2 objectives
// - Completed: 12 objectives
// - Blocked: 1 objectives
// Total Active: 10 | Total General: 23
```

### Objective Example

```markdown
#### Optimize LLM Token Usage
*ID: `perf-2026-q1`* | **Status:** in-progress | **Created:** 2026-02-09

Reduce token consumption by implementing context caching and smart compression.

**Deadline:** 2026-03-31

**Dependencies:** `cache-2026-q1`

**Tags:** `performance`, `cost`
```

---

## 🏥 Health Check - Reflective Monitoring

### What Is It?

The Health Checker monitors the "health" of the project:
- **Build Status** - Does the project compile?
- **Test Status** - Are tests passing?
- **Git Status** - Uncommitted changes?
- **Task Status** - How many pending tasks?
- **Recommendations** - Suggestions for action

### Why Is It Important?

- 🚨 **Early Detection** - Identifies issues before they grow
- 🔄 **Auto-Correction** - Agent can fix broken builds
- 📊 **Visibility** - Clear project status
- 💡 **Proactive** - Actionable recommendations

### How Does It Work?

#### Heartbeat Integration

Health Check runs automatically during Heartbeat:

```
💓 Heartbeat #5 at 2026-02-09 14:30:00
Health: Build=passing, Tests=passing, Git=clean, Tasks=3
```

#### Automatic Build Detection

The checker detects the project type:
- **Go** → `go build ./...`
- **Node.js** → `npm run build`
- **Python** → `python setup.py build`
- **Rust** → `cargo build`

#### Possible Statuses

- ✅ **passing** - Everything OK
- ❌ **failing** - Problem detected
- ⚪ **skipped** - Not applicable
- ❔ **unknown** - Not verified

### Heartbeat with Reflection

The new Heartbeat combines:
1. **Health Check** - Checks project state
2. **Task Check** - Searches for pending tasks
3. **Reflection** - Decides if action is needed

#### Generated Prompt Example

```markdown
🔔 **Heartbeat Execution** - 2026-02-09 14:30:00

## 🏥 Project Health Status

- **Build:** passing
- **Tests:** failing
- **Git:** uncommitted changes (5 uncommitted files)
- **Pending Tasks:** 3

⚠️ **Warnings:**
- Tests are failing

💡 **Recommendations:**
- 🧪 Address failing tests to maintain code quality

## 📋 Your Actions

⚠️ **PRIORITY:** Tests are failing. Please address test failures.

Respond with:
1. If you took action: Brief summary of what was done
2. If no action needed: Just say 'NO PENDING TASKS'
```

### Deep Reflection (Every 5 Heartbeats)

Deeper strategic analysis:

```go
// Automatically executed every 5 heartbeats
performDeepReflection(healthStatus)
```

Analyzes:
- ✅ Recent learnings (brain.md)
- ✅ Strategic objectives (roadmap.md)
- ✅ Recurring patterns or issues
- ✅ Recommendations for improvements

### Public API

```go
// Get health checker
checker := agent.GetHealthChecker()

// Run health check
status := agent.PerformHealthCheck()

// Access results
fmt.Println("Build:", status.BuildStatus)
fmt.Println("Tests:", status.TestStatus)
fmt.Println("Git:", status.GitStatus)
fmt.Println("Pending Tasks:", status.PendingTasks)

// Generate formatted report
report := checker.FormatReport(status)
```

---

## 🔄 Integrated Workflow

### Scenario 1: Broken Build

1. **Heartbeat detects** build failing
2. **Health Check** identifies specific error
3. **Agent wakes up** with high priority
4. **Brain records** error and solution after fix
5. **Roadmap updates** if it affected strategic goal

### Scenario 2: Objective Completed

1. **Agent completes** major task
2. **Brain records** technical decision made
3. **Roadmap marks** objective as completed
4. **Health Check** confirms quality (tests passing)
5. **Deep Reflection** suggests next objective

---

## ⚙️ Configuration

### Config.json

```json
{
  "heartbeat_interval": 300,
  "test_command": "go test ./...",
  "run_tests_before_apply": true
}
```

### Disable Heartbeat

```json
{
  "heartbeat_interval": 0
}
```

---

## 🎯 Best Practices

### Brain
1. ✅ **Review regularly** - Read brain.md periodically
2. ✅ **Edit manually** - Add your own lessons
3. ✅ **Use tags** - Easier future search
4. ✅ **Be specific** - Context matters

### Roadmap
1. ✅ **Keep updated** - Review objectives monthly
2. ✅ **Prioritize** - Not everything can be high priority
3. ✅ **Clear dependencies** - Avoid blockers
4. ✅ **Celebrate achievements** - Mark as completed

### Health Check
1. ✅ **Proper interval** - 5-10 minutes is ideal
2. ✅ **Monitor logs** - See what the agent detects
3. ✅ **Trust but verify** - Health check complements, doesn't replace
4. ✅ **Adjust test command** - For your specific project

---

## 🎓 Conclusion

These three features transform the agent from a tactical executor into a **strategic partner**:
- 🧠 **Brain** - Learns from experience
- 🗺️ **Roadmap** - Plans for the future
- 🏥 **Health** - Maintains quality

Together, they create a virtuous cycle of **execution → learning → planning → continuous improvement**.
