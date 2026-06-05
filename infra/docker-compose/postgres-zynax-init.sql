-- SPDX-License-Identifier: Apache-2.0
-- Creates per-service databases in the zynax Postgres instance.
-- Each service owns its schema exclusively (ADR-008).
-- Runs once on first container start via docker-entrypoint-initdb.d.
CREATE DATABASE task_broker;
CREATE DATABASE agent_registry;
