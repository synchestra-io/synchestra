# Feature: UI / Web App

**Status:** Conceptual

## Summary

The web app is Synchestra's progressive web application for human users. It provides project navigation, feature browsing, proposal creation, task creation/enqueueing, and worker visibility in a browser-accessible interface built with TypeScript, Nx, Angular, and Ionic.

## Problem

Users need a graphical interface for Synchestra that works well in browsers, can behave like an installable app, and supports the most important project-management flows without dropping to raw repository files or low-level CLI commands.

## Proposed Behavior

### Delivery model

The web app is a progressive web app.

Its implementation stack is:

- TypeScript
- Nx
- Angular
- Ionic

### Home page

The app starts at a home page showing the list of projects the current user is working with.

Each project entry opens the selected project's menu.

### Project menu

The initial project menu contains:

- `Features`
- `Tasks`
- `Workers`

This menu is the starting point for all MVP flows in the web app.

### Features flow

The `Features` screen shows the root features of the selected project.

Selecting a feature opens that feature's screen. The feature screen must include:

- The feature content
- The feature's proposals list
- A button or action to open the `New proposal` screen

The `New proposal` screen creates a proposal for that feature and may also create and link a GitHub issue for MVP.

### Tasks flow

The `Tasks` screen shows the root tasks of the selected project.

For MVP, the screen supports:

- Viewing root tasks
- Creating a task
- Enqueueing a task

Task creation and enqueueing must respect the canonical task rules already defined elsewhere in Synchestra's specifications.

### Workers flow

The `Workers` screen shows workers available to execute tasks and run project validation.

Examples include:

- VMs
- Docker-based workers
- Cloud workers
- Local workers

For MVP, this screen is a visibility and selection surface. Provisioning and advanced worker orchestration are outside this subfeature's initial scope.

## Additional Rules

- The web app must clearly preserve the distinction between current feature specification and non-normative proposals.
- The web app should present proposals and task actions at the feature/project level where the user already has relevant context.

## Outstanding Questions

None at this time.
