resource "random_id" "bucket_prefix" {
  byte_length = 8
}

resource "google_service_account" "cloud_functions" {
  account_id = local.service_name
}

resource "google_project_iam_member" "token_creator" {
  project = data.google_project.default.project_id
  role    = "roles/iam.serviceAccountTokenCreator"
  member  = "serviceAccount:${google_service_account.cloud_functions.email}"
}

resource "google_project_iam_member" "datastore_user" {
  project = data.google_project.default.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.cloud_functions.email}"
}

resource "google_service_account_iam_member" "default_compute" {
  service_account_id = data.google_compute_default_service_account.default.id
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.cloud_functions.email}"
}

resource "google_pubsub_topic" "default" {
  name = local.service_name
}

resource "google_cloud_scheduler_job" "default" {
  name     = local.service_name
  schedule = "*/5 * * * *"

  pubsub_target {
    # topic.id is the topic's full resource name.
    topic_name = google_pubsub_topic.default.id
    data       = base64encode("test")
  }
}

resource "google_storage_bucket" "default" {
  name     = "${random_id.bucket_prefix.hex}-${local.service_name}" # Every bucket name must be globally unique
  location = "ASIA-NORTHEAST1"
}

data "archive_file" "default" {
  type        = "zip"
  output_path = "../function-source.zip"
  source_dir  = "../"
  excludes    = ["infra", ".env", "README.md", "LICENSE", ".gitignore"]
}

resource "google_storage_bucket_object" "default" {
  name   = "function-source.zip"
  bucket = google_storage_bucket.default.name
  source = data.archive_file.default.output_path # Path to the zipped function source code
}

resource "google_cloudfunctions2_function" "default" {
  name = local.service_name

  build_config {
    runtime     = "go120"
    entry_point = "MyCloudEventFunction" # Set the entry point
    source {
      storage_source {
        bucket = google_storage_bucket.default.name
        object = google_storage_bucket_object.default.name
      }
    }
  }

  service_config {
    max_instance_count             = 3
    min_instance_count             = 1
    available_memory               = "256M"
    timeout_seconds                = 120
    ingress_settings               = "ALLOW_INTERNAL_ONLY"
    all_traffic_on_latest_revision = true
    service_account_email          = google_service_account.cloud_functions.email
  }

  event_trigger {
    trigger_region = local.region
    event_type     = "google.cloud.pubsub.topic.v1.messagePublished"
    pubsub_topic   = google_pubsub_topic.default.id
    retry_policy   = "RETRY_POLICY_RETRY"
  }
}