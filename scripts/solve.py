import pandas as pd
import random

from pandas import DataFrame

# games

games = pd.read_csv('.data/games.csv')
games_columns = ['AppID', 'Name', 'Windows', 'Mac', 'Linux', 'Genres', 'Release date', 'Average playtime forever', 'Positive', 'Negative']
games = games.filter(games_columns, axis="columns").dropna()
games["Genres"] = games["Genres"].str.lower()

print("Games:", games.shape[0])

games_indie: DataFrame = games[games["Genres"].str.contains("indie")] # type: ignore
print("Games Indie:", games_indie.shape[0])
games_shooter = games[games["Genres"].str.contains("action")]
print("Games Shooter:", games_shooter.shape[0])
games_indie_2010: DataFrame = games_indie[games_indie["Release date"].str.contains("201")] # type: ignore
print("Games Indie 2010:", games_indie_2010.shape[0])

# Q1

q1_count = games[["Linux", "Mac", "Windows"]].sum()
q1_count.to_csv(".results/1.py.csv", header=False, index=True)

# Q2

q1_games = games_indie_2010.filter(["AppID", "Name", "Average playtime forever"], axis="columns")
q1_top = q1_games.sort_values("Average playtime forever", ascending=False).head(10)
q1_top.to_csv(".results/2.py.csv", header=True, index=False)

# reviews

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
    # random for debugging
    if random.random() < 0.978388118:
        return "en"
    else:
        return "es"
    # language, _ = langid.classify(texto)
    # return language

reviews_negative_ingles = reviews_negative.copy()
reviews_negative_ingles["review_language"] = reviews_negative_ingles['review_text'].apply(detect_language)

reviews_negative_ingles = reviews_negative_ingles[reviews_negative_ingles["review_language"] == "en"]
print("Reviews Negative Ingles:", reviews_negative_ingles.shape[0])

# utils

def group(games, reviews) -> DataFrame:
    return pd.merge(games, reviews, left_on='AppID', right_on='app_id', how='inner').groupby("AppID").aggregate({
        "Name": lambda x: ",".join(x.unique()),
        "review_text": "count",
    }).rename({
        "review_text": "Reviews"
    }, axis='columns') # type: ignore

# Q3

q3_grouped = group(games_indie, reviews_positive)
q3_top = q3_grouped.sort_values("Reviews", ascending=False).head(5)
q3_top.to_csv(".results/3.py.csv", header=True, index=True)

# Q4

q4_grouped = group(games_shooter, reviews_negative_ingles)
q4_filtered = q4_grouped[q4_grouped["Reviews"] > 5000]
q4_filtered.sort_index().to_csv(".results/4.py.csv", header=True, index=True)

# Q5

q5_grouped = group(games_shooter, reviews_negative)
percentile = q5_grouped["Reviews"].quantile(0.90)
q5_top = q5_grouped[q5_grouped["Reviews"] >= percentile]
q5_top.sort_index().to_csv(".results/5.py.csv", header=True, index=True)
