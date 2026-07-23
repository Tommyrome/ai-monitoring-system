"""
CentroidTracker
================
Tracker leggero basato sul centroide del bounding box.

Non e' sofisticato quanto SORT/DeepSORT (che usano filtri di Kalman +
Hungarian algorithm per l'assegnazione ottimale), ma e' sufficiente per
una demo e per spiegare il *concetto* di object tracking in un colloquio:

1. Ad ogni nuovo rilevamento si calcola il centroide del bounding box.
2. Si confronta con i centroidi degli oggetti gia' tracciati nel frame
   precedente (distanza euclidea).
3. Se la distanza e' sotto una soglia, l'ID esistente viene mantenuto.
4. Se un oggetto tracciato non appare per troppi frame consecutivi,
   viene rimosso (persona uscita dal campo visivo).

Per portare il progetto a livello "production" si puo' sostituire questa
classe con una vera implementazione SORT/DeepSORT (vedi README, sezione
"Estensioni future").
"""

from collections import OrderedDict
import numpy as np


class CentroidTracker:
    def __init__(self, max_disappeared=30, max_distance=80):
        self.next_object_id = 0
        self.objects = OrderedDict()       # object_id -> centroid (x, y)
        self.bboxes = OrderedDict()        # object_id -> (x1, y1, x2, y2)
        self.disappeared = OrderedDict()   # object_id -> frame count assente
        self.max_disappeared = max_disappeared
        self.max_distance = max_distance

    def _register(self, centroid, bbox):
        self.objects[self.next_object_id] = centroid
        self.bboxes[self.next_object_id] = bbox
        self.disappeared[self.next_object_id] = 0
        self.next_object_id += 1

    def _deregister(self, object_id):
        del self.objects[object_id]
        del self.bboxes[object_id]
        del self.disappeared[object_id]

    def update(self, rects):
        """
        rects: lista di bounding box [(x1, y1, x2, y2), ...] per il frame corrente
        Ritorna: dict {object_id: (centroid, bbox)}
        """
        if len(rects) == 0:
            for object_id in list(self.disappeared.keys()):
                self.disappeared[object_id] += 1
                if self.disappeared[object_id] > self.max_disappeared:
                    self._deregister(object_id)
            return self._as_result()

        input_centroids = np.zeros((len(rects), 2), dtype="int")
        for i, (x1, y1, x2, y2) in enumerate(rects):
            cx = int((x1 + x2) / 2.0)
            cy = int((y1 + y2) / 2.0)
            input_centroids[i] = (cx, cy)

        if len(self.objects) == 0:
            for i in range(len(input_centroids)):
                self._register(input_centroids[i], rects[i])
        else:
            object_ids = list(self.objects.keys())
            object_centroids = list(self.objects.values())

            D = np.linalg.norm(
                np.array(object_centroids)[:, np.newaxis] - input_centroids, axis=2
            )

            rows = D.min(axis=1).argsort()
            cols = D.argmin(axis=1)[rows]

            used_rows, used_cols = set(), set()

            for row, col in zip(rows, cols):
                if row in used_rows or col in used_cols:
                    continue
                if D[row, col] > self.max_distance:
                    continue
                object_id = object_ids[row]
                self.objects[object_id] = input_centroids[col]
                self.bboxes[object_id] = rects[col]
                self.disappeared[object_id] = 0
                used_rows.add(row)
                used_cols.add(col)

            unused_rows = set(range(D.shape[0])) - used_rows
            unused_cols = set(range(D.shape[1])) - used_cols

            for row in unused_rows:
                object_id = object_ids[row]
                self.disappeared[object_id] += 1
                if self.disappeared[object_id] > self.max_disappeared:
                    self._deregister(object_id)

            for col in unused_cols:
                self._register(input_centroids[col], rects[col])

        return self._as_result()

    def _as_result(self):
        return {
            oid: (self.objects[oid], self.bboxes[oid]) for oid in self.objects
        }
