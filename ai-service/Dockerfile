FROM python:3.11-slim

RUN apt-get update && apt-get install -y \
    libgl1 \
    libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Nota: l'accesso alla webcam da dentro un container richiede il passaggio
# esplicito del device (--device=/dev/video0) o l'uso di uno stream RTSP.
# Vedi README per i dettagli.
CMD ["python", "detector.py"]
