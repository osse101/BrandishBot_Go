import sys
import os
import hashlib
from PIL import Image, ImageDraw

def get_color(prompt):
    prompt = prompt.lower()
    if "rusty" in prompt or "brown" in prompt:
        return (139, 69, 19, 255) # SaddleBrown
    if "silver" in prompt or "basic" in prompt or "grey" in prompt:
        return (192, 192, 192, 255) # Silver
    if "golden" in prompt or "gold" in prompt or "coin" in prompt:
        return (255, 215, 0, 255) # Gold
    if "ray gun" in prompt:
        return (255, 0, 0, 255) # Red
    if "blacksmith" in prompt:
        return (105, 105, 105, 255) # DimGray
    if "explorer" in prompt:
        return (34, 139, 34, 255) # ForestGreen
    if "merchant" in prompt:
        return (70, 130, 180, 255) # SteelBlue
    if "gambler" in prompt:
        return (128, 0, 128, 255) # Purple
    if "farmer" in prompt:
        return (154, 205, 50, 255) # YellowGreen
    if "scholar" in prompt:
        return (65, 105, 225, 255) # RoyalBlue

    # Fallback: Generate color from hash
    h = hashlib.md5(prompt.encode()).hexdigest()
    r = int(h[0:2], 16)
    g = int(h[2:4], 16)
    b = int(h[4:6], 16)
    return (r, g, b, 255)

def draw_shape(draw, shape_type, color):
    if shape_type == "box":
        draw.rectangle([4, 4, 27, 27], fill=color, outline=(0, 0, 0, 255))
        draw.rectangle([6, 6, 25, 25], fill=None, outline=(255, 255, 255, 100)) # Highlight
    elif shape_type == "circle":
        draw.ellipse([4, 4, 27, 27], fill=color, outline=(0, 0, 0, 255))
        draw.ellipse([8, 8, 20, 20], fill=None, outline=(255, 255, 255, 100)) # Highlight
    elif shape_type == "gun":
        draw.polygon([(8, 10), (24, 10), (24, 16), (14, 16), (14, 24), (8, 24)], fill=color, outline=(0, 0, 0, 255))
    elif shape_type == "person":
        # Head
        draw.ellipse([10, 4, 22, 16], fill=(255, 220, 177, 255), outline=(0, 0, 0, 255))
        # Body
        draw.rectangle([8, 16, 24, 30], fill=color, outline=(0, 0, 0, 255))
    else:
        # Default square
        draw.rectangle([8, 8, 24, 24], fill=color, outline=(0, 0, 0, 255))

def get_shape(prompt):
    prompt = prompt.lower()
    if "lootbox" in prompt or "box" in prompt:
        return "box"
    if "coin" in prompt or "money" in prompt:
        return "circle"
    if "gun" in prompt or "blaster" in prompt:
        return "gun"
    if any(x in prompt for x in ["blacksmith", "explorer", "merchant", "gambler", "farmer", "scholar"]):
        return "person"
    return "default"

def generate_image(output_path, prompt):
    img = Image.new('RGBA', (32, 32), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    color = get_color(prompt)
    shape = get_shape(prompt)

    draw_shape(draw, shape, color)

    # Add "pixel art" noise/texture (simplistic)
    # Actually, keep it clean for "Hi-Bit Chibi"

    # Save
    img.save(output_path)
    print(f"Generated {output_path} with prompt '{prompt}' (Shape: {shape}, Color: {color})")

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python pixel_pusher_gen.py <output_path> <prompt>")
        sys.exit(1)

    output_path = sys.argv[1]
    prompt = sys.argv[2]

    generate_image(output_path, prompt)
