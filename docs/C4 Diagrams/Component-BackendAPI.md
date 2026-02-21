# Component Diagram - Backend API (Complete)

```mermaid

C4Component
title Component Diagram - Backend API (Complete)
ContainerDb(db, "PostgreSQL", "Relational Database", "Stores persistent data")
Container_Boundary(api, "Backend API") {
    Boundary(presentation, "Presentation Layer (Controllers)") {
        Component(schedule_ctrl, "Schedule Controller", "REST API", "Endpoints for Organizers to upload/edit tracks & sessions")
        Component(booking_ctrl, "Booking Controller", "REST API", "Endpoints for Attendees to book spots")
        Component(checkin_ctrl, "Check-in Controller", "REST API", "Endpoints for Organizers to scan tickets")
        Component(ws_handler, "WebSocket Handler", "Socket.io/WS", "Push channel for live updates")
    }
    Boundary(core, "Application Core (Use Cases)") {
        Component(manage_uc, "ManageScheduleUseCase", "Use Case", "Validates track times, saves sessions, notifies changes.")
        Component(book_uc, "BookSessionUseCase", "Use Case", "Checks capacity, creates booking.")
        Component(walkin_uc, "WalkInEntryUseCase", "Use Case", "Allows entry if session active & seats released.")
        Component(checkin_uc, "ProcessCheckInUseCase", "Use Case", "Marks attendee present, triggers count update.")
        Component(release_uc, "ReleaseNoShowsUseCase", "Use Case", "Cancels bookings X mins after start.")
    }
    Boundary(infra, "Infrastructure Layer") {
        Component(session_repo, "SessionRepository", "Repo Impl", "SQL for Tracks/Sessions")
        Component(booking_repo, "BookingRepository", "Repo Impl", "SQL for Bookings")
        Component(scheduler, "Job Scheduler", "Cron/Quartz", "Triggers background tasks")
        Component(rt_gateway, "RealTimeGateway", "Adapter", "Abstracts WebSocket logic")
    }
}
Rel(schedule_ctrl, manage_uc, "Uploads/Edits Schedule")
Rel(manage_uc, session_repo, "Persists Sessions")
Rel(manage_uc, rt_gateway, "Broadcasts 'Schedule Changed'")
Rel(booking_ctrl, book_uc, "Requests Booking")
Rel(book_uc, booking_repo, "Saves Booking")
Rel(book_uc, session_repo, "Checks Capacity")
Rel(checkin_ctrl, checkin_uc, "Scans Ticket")
Rel(checkin_ctrl, walkin_uc, "Registers Walk-in")
Rel(checkin_uc, booking_repo, "Updates Status")
Rel(checkin_uc, rt_gateway, "Broadcasts 'Count +1'")
Rel(scheduler, release_uc, "Triggers (Time X)")
Rel(release_uc, booking_repo, "Cancels No-Shows")
Rel(release_uc, rt_gateway, "Broadcasts 'Seats Released'")
Rel(rt_gateway, ws_handler, "Push Events")
Rel(session_repo, db, "SQL")
Rel(booking_repo, db, "SQL")
```
