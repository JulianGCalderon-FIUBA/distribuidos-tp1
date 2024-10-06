import numpy as np
import pandas as pd
import os
import langid
import time

from pandas import DataFrame

games = pd.read_csv('.data/games.csv')
games_columns = ['AppID', 'Name', 'Windows', 'Mac', 'Linux', 'Genres', 'Release date', 'Average playtime forever', 'Positive', 'Negative']
games = games.filter(games_columns, axis="columns").dropna()
games["Genres"] = games["Genres"].str.lower()

print("Games:", games.shape[0])

games_indie: DataFrame = games[games["Genres"].str.contains("indie")] # type: ignore
print("Games Indie:", games_indie.shape[0])
games_shooter = games[games["Genres"].str.contains("action")]
print("Games Shooter:", games_shooter.shape[0])
games_indie_2010 = games_indie[games_indie["Release date"].str.contains("201")]
print("Games Indie 2010:", games_indie_2010.shape[0])

reviews = pd.read_csv('.data/reviews.csv')
reviews_columns = ['app_id', 'review_text', 'review_score']
reviews = reviews.filter(reviews_columns, axis="columns").dropna()
reviews['review_text'] = reviews['review_text'].astype(str)

print("Reviews:", reviews.shape[0])

reviews_positive = reviews[reviews["review_score"] == 1]
print("Reviews Positive:", reviews_positive.shape[0])
reviews_negative: DataFrame = reviews[reviews["review_score"] == -1] # type: ignore
print("Reviews Negative:", reviews_negative.shape[0])

def detect_language(texto):
    language, _ = langid.classify(texto)
    return language

reviews_negative_ingles = reviews_negative.copy()
reviews_negative_ingles["review_language"] = reviews_negative_ingles['review_text'].apply(detect_language)

reviews_negative_ingles = reviews_negative_ingles[reviews_negative_ingles["review_language"] == "en"]
print("Reviews Negative Ingles:", reviews_negative_ingles.shape[0])
