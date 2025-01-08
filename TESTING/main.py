import os
import nltk
from nltk.sentiment import SentimentIntensityAnalyzer

def analyze_sentiment(text: str) -> dict:
    nltk.download('vader_lexicon', quiet=True)
    sia = SentimentIntensityAnalyzer()
    return sia.polarity_scores(text)

def main():
    text = os.getenv('PARAM', 'Texto de prueba')
    
    sentiment = analyze_sentiment(text)
    
    print(f"Análisis de sentimiento para: '{text}'")
    
    if sentiment['compound'] >= 0.05:
        print("Sentimiento: Positivo")
    elif sentiment['compound'] <= -0.05:
        print("Sentimiento: Negativo")
    else:
        print("Sentimiento: Neutral")

if __name__ == "__main__":
    main() 