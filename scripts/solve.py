import pandas as pd
import os
import argparse

from pandas import DataFrame

parser = argparse.ArgumentParser()
parser.add_argument('input')
parser.add_argument('output')
args = parser.parse_args()

# games

games = pd.read_csv(f'{args.input}/games.csv')
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

try:
    os.mkdir(".py-results")
except:
    pass

# Q1

q1_count = games[["Linux", "Mac", "Windows"]].sum().to_frame().transpose()
q1_count.to_csv(f"{args.output}/1.csv", header=True, index=False)

# Q2

q1_games = games_indie_2010.filter(["AppID", "Name", "Average playtime forever"], axis="columns")
q1_top = q1_games.sort_values("Average playtime forever", ascending=False).head(10)
q1_top.to_csv(f"{args.output}/2.csv", header=True, index=False)

# reviews

reviews = pd.read_csv(f'{args.input}/reviews.csv')
reviews_columns = ['app_id', 'review_text', 'review_score']
reviews = reviews.filter(reviews_columns, axis="columns").dropna()
reviews['review_text'] = reviews['review_text'].astype(str)

print("Reviews:", reviews.shape[0])

reviews_positive = reviews[reviews["review_score"] == 1]
print("Reviews Positive:", reviews_positive.shape[0])
reviews_negative: DataFrame = reviews[reviews["review_score"] == -1] # type: ignore
print("Reviews Negative:", reviews_negative.shape[0])

# We use another dataset, which was filtered in Go
reviews_negative_ingles = pd.read_csv(f'{args.input}/reviews-english-negative.csv')
reviews_negative_ingles = reviews_negative_ingles.filter(reviews_columns, axis="columns").dropna()
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
q3_top.to_csv(f"{args.output}/3.csv", header=True, index=True)

# Q4

q4_grouped = group(games_shooter, reviews_negative_ingles)
q4_filtered = q4_grouped[q4_grouped["Reviews"] > 5000]
q4_filtered.sort_index().to_csv(f"{args.output}/4.csv", header=True, index=True)

# Q5

q5_grouped = group(games_shooter, reviews_negative)
percentile = q5_grouped["Reviews"].quantile(0.90)
q5_top = q5_grouped[q5_grouped["Reviews"] >= percentile]
q5_top.sort_index().to_csv(f"{args.output}/5.csv", header=True, index=True)
