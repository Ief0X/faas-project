import os
import sys
import json
import nltk
from nltk.sentiment import SentimentIntensityAnalyzer

def analyze_sentiment(text: str) -> dict:
    nltk.download('vader_lexicon', quiet=True)
    sia = SentimentIntensityAnalyzer()
    return sia.polarity_scores(text)

def main():
    # Intentar leer el parámetro de diferentes fuentes
    param = None
    
    # 1. Intentar leer de la variable de entorno
    param = os.getenv('PARAM')
    
    # 2. Si está vacío, intentar leer de stdin
    if not param:
        try:
            input_data = sys.stdin.read()
            if input_data:
                try:
                    param_json = json.loads(input_data)
                    param = param_json.get('param', '')
                except json.JSONDecodeError:
                    param = input_data.strip()
        except:
            pass
    
    # 3. Si aún está vacío, usar valor por defecto
    if not param:
        param = 'Texto de prueba'
    
    sentiment = analyze_sentiment(param)
    
    print(f"Análisis de sentimiento para: '{param}'")
    
    if sentiment['compound'] >= 0.05:
        print("Sentimiento: Positivo")
    elif sentiment['compound'] <= -0.05:
        print("Sentimiento: Negativo")
    else:
        print("Sentimiento: Neutral")

if __name__ == "__main__":
    main() 