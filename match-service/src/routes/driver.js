const express = require('express');
const { Op } = require('sequelize');
const Driver = require('../models/driver');
const { redisClient } = require('../index');
const { getDistanceFromLatLonInKm } = require('../utils/distance');

const router = express.Router();

// Update driver's ready status and location
router.post('/beacon', async (req, res) => {
  try {
    const { user_id, is_ready, lat, lng } = req.body;

    // Update or create driver status
    const [driver, created] = await Driver.findOrCreate({
      where: { user_id },
      defaults: {
        is_ready,
        current_lat: lat,
        current_lng: lng,
        last_location_update: new Date()
      }
    });

    if (!created) {
      await driver.update({
        is_ready,
        current_lat: lat,
        current_lng: lng,
        last_location_update: new Date()
      });
    }

    // If driver is ready, store location in Redis for quick lookup
    if (is_ready) {
      await redisClient.geoAdd('driver_locations', {
        longitude: lng,
        latitude: lat,
        member: user_id
      });
    } else {
      // Remove from Redis if not ready
      await redisClient.zRem('driver_locations', user_id);
    }

    res.json({ status: 'success', driver });
  } catch (error) {
    console.error('Driver beacon error:', error);
    res.status(500).json({ error: 'Failed to update driver status' });
  }
});

// Get nearby drivers within 1km
router.get('/nearby', async (req, res) => {
  try {
    const { lat, lng } = req.query;

    // Search for nearby drivers using Redis geospatial
    const nearbyDrivers = await redisClient.geoSearch(
      'driver_locations',
      {
        latitude: parseFloat(lat),
        longitude: parseFloat(lng)
      },
      {
        radius: 1,
        unit: 'km'
      }
    );

    // Get full driver details from database
    const drivers = await Driver.findAll({
      where: {
        user_id: { [Op.in]: nearbyDrivers },
        is_ready: true
      }
    });

    res.json({ drivers });
  } catch (error) {
    console.error('Nearby drivers search error:', error);
    res.status(500).json({ error: 'Failed to find nearby drivers' });
  }
});

module.exports = router;