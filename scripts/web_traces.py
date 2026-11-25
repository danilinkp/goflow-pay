#!/usr/bin/env python3
"""Create a successful transfer trace through all five WEB lab services."""

import json
import os
import sys
import time
import urllib.error
import urllib.request
import uuid

BASE = f"http://127.0.0.1:{os.environ.get('NGINX_PORT', '80')}"


def post(path, payload, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(
        BASE + path,
        data=json.dumps(payload).encode(),
        headers=headers,
        method="POST",
    )
    try:
        response = urllib.request.urlopen(request, timeout=10)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        body = response.read().decode()
        data = json.loads(body) if body else {}
        trace_id = response.headers.get("X-Trace-ID", "")
        return response.status, data, trace_id


def require(path, payload, token=None, statuses=(200, 201)):
    status, data, trace_id = post(path, payload, token)
    if status not in statuses or not trace_id:
        raise RuntimeError(f"{path}: HTTP {status}, Trace ID={trace_id}, response={data}")
    return data, trace_id


def bank_account_number(bic):
    prefix = "40702643" + "0" * 11
    weights = (7, 1, 3) * 7 + (7, 1)
    for digit in "0123456789":
        number = prefix + digit
        checksum = sum(int(value) * weight % 10 for value, weight in zip(bic[-3:] + number, weights))
        if checksum % 10 == 0:
            return number
    raise RuntimeError("could not generate bank account number")


def wait_for_kafka_spans(trace_id):
    url = f"{BASE}/monitoring/api/datasources/proxy/uid/tempo/api/traces/{trace_id}"
    for _ in range(25):
        time.sleep(2)
        try:
            with urllib.request.urlopen(url, timeout=5) as response:
                trace = json.load(response)
        except (urllib.error.URLError, TimeoutError, ValueError):
            continue
        names = {
            span.get("name", "")
            for batch in trace.get("batches", [])
            for scope in batch.get("scopeSpans", [])
            for span in scope.get("spans", [])
        }
        if any(name.startswith("publish ") for name in names) and any(
                name.startswith("consume ") for name in names
        ):
            return True
    return False


def wait_for_ready():
    url = f"{BASE}/ready"
    for _ in range(15):
        try:
            with urllib.request.urlopen(url, timeout=4) as response:
                if response.status == 200:
                    return
        except (urllib.error.URLError, TimeoutError):
            pass
        time.sleep(1)
    raise RuntimeError("gateway or its gRPC dependencies are not ready")


def main():
    wait_for_ready()
    suffix = uuid.uuid4().hex[:12]
    registration, registration_trace = require(
        "/api/v1/auth/register/company",
        {
            "login": f"trace-demo-{suffix}",
            "email": f"trace-demo-{suffix}@example.test",
            "password": uuid.uuid4().hex,
            "company_name": f"Trace demo {suffix}",
        },
    )
    token = registration["token"]
    print(f"Регистрация (gateway → auth): {registration_trace}", flush=True)

    source, _ = require("/api/v1/accounts", {"currency": "RUB"}, token)
    destination, _ = require("/api/v1/accounts", {"currency": "RUB"}, token)
    bic = "044525225"
    bank, _ = require(
        "/api/v1/accounts/bank",
        {
            "account_id": source["account_id"],
            "name": "Trace demo bank",
            "bic": bic,
            "settlement_account": bank_account_number(bic),
            "currency": "RUB",
        },
        token,
    )

    for attempt in range(5):
        status, deposit, deposit_trace = post(
            "/api/v1/accounts/bank/deposit",
            {
                "account_id": source["account_id"],
                "bank_account_id": bank["bank_account_id"],
                "amount": 100,
                "idempotency_key": str(uuid.uuid4()),
            },
            token,
        )
        if status == 200 and deposit_trace:
            break
        if "deposit temporarily unavailable" not in str(deposit) or attempt == 4:
            raise RuntimeError(f"deposit: HTTP {status}, response={deposit}")
    print(f"Пополнение (gateway → auth → accounts → Kafka → notifications): {deposit_trace}", flush=True)

    transfer, transfer_trace = require(
        "/api/v1/transactions/transfer",
        {
            "from_account_id": source["account_id"],
            "to_account_id": destination["account_id"],
            "amount": 10,
            "currency": "RUB",
            "idempotency_key": str(uuid.uuid4()),
        },
        token,
    )
    print(f"Перевод (gateway → auth → transactions → accounts → Kafka → notifications): {transfer_trace}", flush=True)
    print(f"ID перевода: {transfer['transaction_id']}", flush=True)
    if not wait_for_kafka_spans(transfer_trace):
        print("Kafka-спаны пока не найдены в Tempo; проверьте трассу через несколько секунд.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, KeyError, ValueError) as error:
        print(f"Не удалось создать демо-трассу: {error}", file=sys.stderr)
        sys.exit(1)
