CREATE TABLE IF NOT EXISTS public.categories (
    Categories JSON
);

INSERT INTO public.categories 
    (Categories)
VALUES
    ('{"categories":[{"name":"Временная категория", "subcategories":["Временная подкатегория 1", "Временная подкатегория 2"]}]}');