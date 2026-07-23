"""
Zone monitoring & anomaly detection
====================================
Logica per:
  - definire zone virtuali (poligoni normalizzati 0..1) su ogni telecamera
  - capire se un centroide persona si trova dentro una zona
  - generare eventi di tipo "zone_alert" o "anomaly"

Le zone vengono definite in config.py per semplicita' della demo, ma in
un sistema reale andrebbero caricate dal backend (tabella `zones`).
"""

from datetime import datetime, time as dtime
import cv2
import numpy as np


class Zone:
    def __init__(self, name, polygon_norm, zone_type="restricted"):
        """
        polygon_norm: lista di punti [(x, y), ...] con coordinate normalizzate 0..1
        zone_type: "restricted" (zona vietata) | "counting" (conteggio ingressi/uscite)
        """
        self.name = name
        self.polygon_norm = polygon_norm
        self.zone_type = zone_type

    def polygon_pixels(self, frame_w, frame_h):
        return np.array(
            [(int(x * frame_w), int(y * frame_h)) for x, y in self.polygon_norm],
            dtype=np.int32,
        )

    def contains(self, point, frame_w, frame_h):
        poly = self.polygon_pixels(frame_w, frame_h)
        return cv2.pointPolygonTest(poly, (float(point[0]), float(point[1])), False) >= 0


class AnomalyDetector:
    """
    Regole semplici di anomaly detection, pensate per essere spiegate
    facilmente in un colloquio tecnico:

      1. Presenza fuori orario consentito (es. nessuno dovrebbe essere
         in ufficio tra le 22:00 e le 06:00).
      2. Troppe persone contemporaneamente in una zona (overcrowding).
      3. Permanenza troppo lunga di una persona nella stessa zona
         (dwell time eccessivo, es. > 5 minuti in un'area server).
    """

    def __init__(self, allowed_start=dtime(6, 0), allowed_end=dtime(22, 0),
                 max_people_per_zone=5, max_dwell_seconds=300):
        self.allowed_start = allowed_start
        self.allowed_end = allowed_end
        self.max_people_per_zone = max_people_per_zone
        self.max_dwell_seconds = max_dwell_seconds
        self._entry_times = {}  # (zone_name, track_id) -> datetime primo ingresso

    def check_out_of_hours(self, now=None):
        now = now or datetime.now()
        t = now.time()
        if self.allowed_start <= self.allowed_end:
            in_hours = self.allowed_start <= t <= self.allowed_end
        else:  # intervallo che attraversa la mezzanotte
            in_hours = t >= self.allowed_start or t <= self.allowed_end
        return not in_hours

    def check_overcrowding(self, people_in_zone_count):
        return people_in_zone_count > self.max_people_per_zone

    def check_dwell_time(self, zone_name, track_id, now=None):
        now = now or datetime.now()
        key = (zone_name, track_id)
        if key not in self._entry_times:
            self._entry_times[key] = now
            return False
        elapsed = (now - self._entry_times[key]).total_seconds()
        return elapsed > self.max_dwell_seconds

    def clear_track(self, zone_name, track_id):
        self._entry_times.pop((zone_name, track_id), None)
