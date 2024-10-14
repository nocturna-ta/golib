CREATE TABLE IF NOT EXISTS retry_event_log(
                                              id                  varchar(36),
    source_topic        varchar(255)    NOT NULL,
    service_name        varchar(255)    NOT NULL,
    data                text,
    status              varchar(50)     NOT NULL,
    count               int8,
    backoff_date        timestamp(6)    NOT NULL DEFAULT current_timestamp(6),
    error               text,
    created_time        timestamp(6)    NOT NULL DEFAULT current_timestamp(6),
    updated_time        timestamp(6)    NOT NULL DEFAULT current_timestamp(6) ON UPDATE current_timestamp(6),
    PRIMARY KEY (id)
    );
CREATE INDEX backoff_date_idx ON retry_event_log (backoff_date);
CREATE INDEX created_time_idx ON retry_event_log (created_time);
CREATE INDEX count_status_backoff_date ON retry_event_log (count, status, backoff_date);