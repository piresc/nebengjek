const { Model, DataTypes } = require('sequelize');
const { sequelize } = require('../config/database');

class Driver extends Model {}

Driver.init({
  id: {
    type: DataTypes.UUID,
    defaultValue: DataTypes.UUIDV4,
    primaryKey: true
  },
  user_id: {
    type: DataTypes.UUID,
    allowNull: false,
    unique: true
  },
  is_ready: {
    type: DataTypes.BOOLEAN,
    defaultValue: false
  },
  current_lat: {
    type: DataTypes.DECIMAL(10, 8),
    allowNull: true
  },
  current_lng: {
    type: DataTypes.DECIMAL(11, 8),
    allowNull: true
  },
  last_location_update: {
    type: DataTypes.DATE,
    defaultValue: DataTypes.NOW
  },
  created_at: {
    type: DataTypes.DATE,
    defaultValue: DataTypes.NOW
  },
  updated_at: {
    type: DataTypes.DATE,
    defaultValue: DataTypes.NOW
  }
}, {
  sequelize,
  modelName: 'Driver',
  tableName: 'drivers',
  timestamps: true,
  underscored: true
});

module.exports = Driver;