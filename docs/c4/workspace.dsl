workspace "Event Booking System" "C4 model for event booking, scheduling, and check-in management." {

    !identifiers hierarchical

    model {
        attendee = person "Event Attendee" "A person attending the multitrack event."
        organizer = person "Event Organizer" "Staff member managing the event logistics."

        eventSystem = softwareSystem "Event Booking System" "Allows booking, scheduling, and check-in management to prevent overcrowding." {
            attendeeApp = container "Attendee Mobile App" "Mobile Framework" "Allows attendees to view tracks and book sessions."
            checkinApp = container "Organizer Check-in App" "Mobile Framework" "Used to scan tickets or check users into specific sessions."
            orgWebapp = container "Organizer Web Portal" "Single Page Application" "Dashboard for loading schedule and session details."
            backendApi = container "Backend API" "REST API" "Centralizes logic, handles booking validation and availability checks." {
                group "Presentation Layer (Controllers)" {
                    scheduleCtrl = component "Schedule Controller" "REST API" "Endpoints for Organizers to upload/edit tracks & sessions"
                    bookingCtrl = component "Booking Controller" "REST API" "Endpoints for Attendees to book spots"
                    checkinCtrl = component "Check-in Controller" "REST API" "Endpoints for Organizers to scan tickets"
                    wsHandler = component "WebSocket Handler" "Socket.io/WS" "Push channel for live updates"
                }
                group "Application Core (Services)" {
                    manageSvc = component "EventService" "Service" "Validates track times, saves sessions, notifies changes."
                    bookSvc = component "BookSessionService" "Service" "Checks capacity, creates booking."
                    walkinSvc = component "WalkInEntryService" "Service" "Allows entry if session active & seats released."
                    checkinSvc = component "ProcessCheckInService" "Service" "Marks attendee present, triggers count update."
                    releaseSvc = component "ReleaseNoShowsService" "Service" "Cancels bookings X mins after start."
                }
                group "Infrastructure Layer" {
                    sessionRepo = component "SessionRepository" "Repo Impl" "SQL for Tracks/Sessions"
                    bookingRepo = component "BookingRepository" "Repo Impl" "SQL for Bookings"
                    scheduler = component "Job Scheduler" "Cron/Quartz" "Triggers background tasks"
                    rtGateway = component "RealTimeGateway" "Adapter" "Abstracts WebSocket logic"
                }
            }
            db = container "Database" "PostgreSQL" "Stores schedules, user profiles, and booking data." {
                tags "Database"
            }
        }

        attendee -> eventSystem "Views availability and books sessions"
        organizer -> eventSystem "Uploads schedule and checks in attendees"
        attendee -> eventSystem.attendeeApp "Uses"
        organizer -> eventSystem.orgWebapp "Uses to manage schedule"
        organizer -> eventSystem.checkinApp "Uses to scan attendees"
        eventSystem.attendeeApp -> eventSystem.backendApi "Makes API calls to" "JSON/HTTPS"
        eventSystem.orgWebapp -> eventSystem.backendApi "Makes API calls to" "JSON/HTTPS"
        eventSystem.checkinApp -> eventSystem.backendApi "Makes API calls to" "JSON/HTTPS"
        eventSystem.backendApi -> eventSystem.db "Reads/Writes" "SQL/TCP"

        eventSystem.backendApi.scheduleCtrl -> eventSystem.backendApi.manageSvc "Uploads/Edits Schedule"
        eventSystem.backendApi.manageSvc -> eventSystem.backendApi.sessionRepo "Persists Sessions"
        eventSystem.backendApi.manageSvc -> eventSystem.backendApi.rtGateway "Broadcasts Schedule Changed"
        eventSystem.backendApi.bookingCtrl -> eventSystem.backendApi.bookSvc "Requests Booking"
        eventSystem.backendApi.bookSvc -> eventSystem.backendApi.bookingRepo "Saves Booking"
        eventSystem.backendApi.bookSvc -> eventSystem.backendApi.sessionRepo "Checks Capacity"
        eventSystem.backendApi.checkinCtrl -> eventSystem.backendApi.checkinSvc "Scans Ticket"
        eventSystem.backendApi.checkinCtrl -> eventSystem.backendApi.walkinSvc "Registers Walk-in"
        eventSystem.backendApi.checkinSvc -> eventSystem.backendApi.bookingRepo "Updates Status"
        eventSystem.backendApi.checkinSvc -> eventSystem.backendApi.rtGateway "Broadcasts Count update"
        eventSystem.backendApi.scheduler -> eventSystem.backendApi.releaseSvc "Triggers at time X"
        eventSystem.backendApi.releaseSvc -> eventSystem.backendApi.bookingRepo "Cancels No-Shows"
        eventSystem.backendApi.releaseSvc -> eventSystem.backendApi.rtGateway "Broadcasts Seats Released"
        eventSystem.backendApi.rtGateway -> eventSystem.backendApi.wsHandler "Push Events"
        eventSystem.backendApi.sessionRepo -> eventSystem.db "SQL"
        eventSystem.backendApi.bookingRepo -> eventSystem.db "SQL"
    }

    views {
        systemContext eventSystem "SystemContext" "System Context Diagram for Event Booking System" {
            include *
            autoLayout lr
        }
        container eventSystem "Containers" "Container Diagram for Event Booking System" {
            include *
            autoLayout lr
        }
        component eventSystem.backendApi "BackendAPI" "Component Diagram - Backend API (Complete)" {
            include *
            autoLayout tb
        }

        styles {
            element "Element" {
                shape RoundedBox
            }
            element "Person" {
                shape Person
            }
            element "Database" {
                shape Cylinder
            }
        }
    }
}
