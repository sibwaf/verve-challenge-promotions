#!/bin/sh

./wait-for-it.sh $DB_HOST:$DB_PORT

./verve-challenge-promotions $@
