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
    param = None
    
    param = os.getenv('PARAM')
    
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
    if not param:
        param = 'Texto de prueba'
    
    sentiment = analyze_sentiment(param)
    
    if sentiment['compound'] >= 0.05:
        print("Texto: ", param, "Sentimiento: Positivo")
    elif sentiment['compound'] <= -0.05:
        print("Texto: ", param, "Sentimiento: Negativo")
    else:
        print("Texto: ", param, "Sentimiento: Neutral")

if __name__ == "__main__":
    main() 