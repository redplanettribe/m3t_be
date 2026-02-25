# Detailed User Flow for Booking a Session

User vs System actions.

```mermaid

flowchart TB
    subgraph User
        direction TB
        U1[Completes physical event check-in]
        U2[Scans QR code with mobile device]
        U3[Views agenda interface]
        U4[Browses sessions by room/time]
        U5[Examines session details<br/>capacity, description, etc.]
        U6[Selects desired session]
        U7[Clicks Book button]
        U8[Confirms replacement]
        U9[Returns to agenda view]
        U10[Sees session as booked]
        U11[Sees updated personal schedule]
    end

    subgraph System
        direction TB
        S1[Validates QR code]
        S2[Authenticates user via passwordless email login<br/>(one-time code + email)]
        S3[Issues JWT and internal user identifier]
        S4[Returns event data]
        S5[Fetches real-time session data]
        S6[Calculates available spots per session]
        S7[Updates occupancy counters in real-time]
        S8[Displays agenda with current availability]
        S9[Validates session availability]
        S10[Checks user's existing bookings for conflicts]
        S11[Identifies conflicting session]
        S12[Displays overlap warning with conflict details]
        S13[Cancels previous booking]
        S14[Updates session capacity for cancelled session]
        S15[Creates new booking]
        S16[Updates session capacity for new session]
        S17[Proceeds with booking]
        S18[Creates booking record]
        S19[Updates session capacity]
        S20[Generates booking confirmation]
        S21[Updates availability and notifies other clients]
        S22[Confirms successful booking to user]
    end

    Start([start]) --> U1 --> U2
    U2 --> S1 --> S2 --> S3 --> S4
    S4 --> U3 --> U4 --> U5
    U5 --> S5 --> S6 --> S7 --> S8
    S8 --> U6 --> U7
    U7 --> S9 --> S10
    S10 --> Overlap{Session overlaps with<br/>existing booking?}
    Overlap -->|Yes| S11 --> S12
    S12 --> UserChoice{User chooses to<br/>replace existing booking?}
    UserChoice -->|Yes| U8 --> S13 --> S14 --> S15 --> S16
    UserChoice -->|No| U9
    Overlap -->|No| S17 --> S18 --> S19 --> S20
    S16 --> S21 --> S22
    S20 --> S21
    S21 --> S22 --> U10 --> U11
    U11 --> More[Browse more sessions]
    More --> U3
    U9 --> U3
    
```
