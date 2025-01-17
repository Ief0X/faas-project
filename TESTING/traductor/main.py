import os
import sys
import json
from deep_translator import GoogleTranslator

def translate_text(text: str, target_lang: str = 'es') -> str:
    try:
        translator = GoogleTranslator(source='auto', target=target_lang)
        return translator.translate(text)
    except Exception as e:
        return f"Error en traducción: {str(e)}"

def main():
    param = None
    target_lang = 'es'
    
    param = os.getenv('PARAM')
    
    if not param:
        try:
            input_data = sys.stdin.read()
            if input_data:
                try:
                    param_json = json.loads(input_data)
                    param = param_json.get('param', '')
                    target_lang = param_json.get('target_lang', 'es')
                except json.JSONDecodeError:
                    param = input_data.strip()
        except:
            pass
    
    if not param:
        param = "testing text"
    
    translated = translate_text(param, target_lang)
    print(f"Texto original: {param}, Traducción: {translated}")

if __name__ == "__main__":
    main()