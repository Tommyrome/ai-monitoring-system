import logging
import requests
from config import EVENTS_ENDPOINT, API_TOKEN

logger = logging.getLogger("event_client")


def send_event(payload: dict, timeout=2.0):
    """
    Invia un evento al backend Go.
    Ritorna True se l'invio ha avuto successo (HTTP 2xx), False altrimenti.
    Gli errori di rete non devono mai bloccare il loop di detection:
    vengono loggati e ignorati (fail-soft).
    """
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_TOKEN}",
    }
    try:
        resp = requests.post(EVENTS_ENDPOINT, json=payload, headers=headers, timeout=timeout)
        if 200 <= resp.status_code < 300:
            logger.info(
                "Evento inviato: track_id=%s tipo=%s",
                payload.get("track_id"),
                resp.json().get("tipo_evento", "?"),
            )
            return True
        logger.warning("Backend ha rifiutato l'evento (%s): %s", resp.status_code, resp.text)
        return False
    except requests.RequestException as e:
        logger.error("Impossibile contattare il backend: %s", e)
        return False
