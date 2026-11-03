import worker as skeleton


def test_worker_reports_in(server, admin, worker, monkeypatch):
    name, password = worker
    monkeypatch.setenv("SHOPPA_URL", server)
    monkeypatch.setenv("SHOPPA_WORKER", name)
    monkeypatch.setenv("SHOPPA_WORKER_PASSWORD", password)

    skeleton.main(once=True)

    res = admin.get("/api/collections/workers/records", params={"filter": f'name = "{name}"'})
    assert res.json()["items"][0]["last_seen"]
